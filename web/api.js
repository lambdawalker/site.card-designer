import { bearer, authenticated } from "./auth.js";
import { catalog, stockBundles } from "./catalog.js";
const packages = new Map(),
  records = new Map();
let discovery;
export async function request(path, method = "GET", data) {
  const token = bearer();
  const response = await fetch("/api/" + path, {
    method,
    headers: {
      ...(data === undefined ? {} : { "Content-Type": "application/json" }),
      ...(token ? { Authorization: "Bearer " + token } : {}),
    },
    body: data === undefined ? undefined : JSON.stringify(data),
  });
  if (response.status === 204) return null;
  const result = await response.json();
  if (!response.ok)
    throw Error(
      result.error ||
        result.detail ||
        "Request failed (" + response.status + ")",
    );
  return result;
}
export const imageURL = (value) => {
  if (!value?.bytes) return "";
  const prefix = value.bytes.slice(0, 16),
    mime = prefix.startsWith("/9j/")
      ? "image/jpeg"
      : prefix.startsWith("UklGR")
        ? "image/webp"
        : prefix.startsWith("R0lGOD")
          ? "image/gif"
          : "image/png";
  return "data:" + mime + ";base64," + value.bytes;
};
export function mapRow(row, path, index, revision) {
  const id = String(row.id ?? path + ":" + index),
    images = Object.fromEntries(
      ["image", "image_transparent"]
        .map((k) => [k, imageURL(row[k])])
        .filter(([, v]) => v),
    );
  let parameters = row.parameters || {};
  if (typeof parameters === "string") {
    try {
      parameters = JSON.parse(parameters);
    } catch {
      parameters = {};
    }
  }
  const record = {
    ...parameters,
    id,
    record_id: id,
    source_id: "photo-id",
    dataset: "lambdaWalker/ds.photo-id",
    source: "lambdaWalker/ds.photo-id",
    name: parameters.full_name || row.id || "Portrait " + (index + 1),
    prompt: row.prompt || "",
    images,
    photo_url: images.image || images.image_transparent || "",
    sample_source: { revision, path, index, id },
  };
  records.set(id, record);
  records.set(revision+":"+path+":"+id,record);
  return record;
}
async function listing() {
  return (discovery ??= request("datasets/photo-id/packages").catch((error) => {
    discovery = null;
    throw error;
  }));
}
async function packageRows(entry, revision) {
  const key = revision + ":" + entry.path;
  if (!packages.has(key)) {
    const promise = (async () => {
      const cache = globalThis.caches
        ? await caches.open("card-designer-packages-v1")
        : null;
      const url = "/__immutable_package/" + encodeURIComponent(key);
      let response = await cache?.match(url),
        data;
      if (response) data = await response.json();
      else {
        data = await request(
          "datasets/photo-id/package?" +
            new URLSearchParams({ path: entry.path, revision }),
        );
        await cache?.put(
          url,
          new Response(JSON.stringify(data), {
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return data.rows.map((row, index) =>
        mapRow(row, entry.path, index, revision),
      );
    })();
    packages.set(key, promise);
    promise.catch(() => packages.delete(key));
  }
  return packages.get(key);
}
async function page(offset, q) {
  const info = await listing();
  let all = [];
  if (q && !packages.size && info.packages.length)
    await packageRows(info.packages[0], info.revision);
  for (const entry of info.packages) {
    const key = info.revision + ":" + entry.path;
    if (q && !packages.has(key)) continue;
    const rows = await packageRows(entry, info.revision);
    all.push(
      ...rows.filter(
        (row) =>
          !q ||
          (row.prompt + " " + row.name + " " + row.id)
            .toLowerCase()
            .includes(q.toLowerCase()),
      ),
    );
    if (all.length >= offset + 24) break;
  }
  return all.slice(offset, offset + 24);
}
export async function selectedPerson(id, source) {
  const key=source?.revision?source.revision+":"+source.path+":"+id:String(id);
  if (records.has(key)) return records.get(key);
  if (source?.revision && source.path) {
    const rows = await packageRows({ path: source.path }, source.revision);
    return (
      rows.find((row) => row.id === String(id)) || rows[source.index] || {}
    );
  }
  return {};
}
export async function api(path, method = "GET", data) {
  const [route, search = ""] = path.split("?"),
    params = new URLSearchParams(search);
  if (route === "artwork") return [];
  if (route === "datasets/status") return { loading: false };
  if (route === "datasets" && method === "GET")
    return (authenticated() ? await request(path) : []).map((row) => ({
      ...row,
      image_fields: ["image", "image_transparent"],
    }));
  if (route.startsWith("datasets") && method !== "GET")
    throw Error("This deployment uses the fixed Synthetic portraits dataset.");
  if (route === "samples")
    return page(Number(params.get("offset") || 0), params.get("q") || "");
  if (route === "samples/random") {
    const info = await listing();
    if (!info.packages.length) throw Error("Dataset has no packages");
    const entry =
      info.packages[
        crypto.getRandomValues(new Uint32Array(1))[0] % info.packages.length
      ];
    const rows = await packageRows(entry, info.revision);
    return rows[crypto.getRandomValues(new Uint32Array(1))[0] % rows.length];
  }
  if (route.startsWith("samples/")) {
    const record = await selectedPerson(decodeURIComponent(route.slice(8)),params.has("source")?JSON.parse(params.get("source")):null);
    const fields={...record,...record.images};
    return {record,fields,field_types:Object.fromEntries(Object.entries(fields).map(([key,value])=>[key,Object.hasOwn(record.images||{},key)?'rawImage':typeof value]))};
  }
  if (route === "templates") return catalog;
  if (route.startsWith("templates/"))
    return catalog.find((item) => item.id === route.slice(10));
  if (route === "bundles" && method === "GET")
    return [...stockBundles, ...(authenticated() ? await request(path) : [])];
  if (route.startsWith("bundles/stock:")) {
    const key = route.slice(8).split("/")[0],
      bundle = structuredClone(stockBundles.find((item) => item.id === key));
    if (method === "GET") return bundle;
    if (route.endsWith("/clone") && method === "POST")
      return request("bundles", "POST", {
        name: data?.name || bundle.name + " copy",
        layouts: data?.layouts || bundle.layouts,
      });
    throw Error("Clone this built-in bundle to save your changes.");
  }
  if (!authenticated())
    throw Error("Sign in to save private bundles and manage API tokens.");
  return request(path, method, data);
}
