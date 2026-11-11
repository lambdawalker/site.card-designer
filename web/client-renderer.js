import bwipjs from "bwip-js/browser";
export function seeded(seed) {
  let a = Number(seed) >>> 0;
  return () => {
    a += 0x6d2b79f5;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const svgURL = (svg) =>
  "data:image/svg+xml;base64," + btoa(unescape(encodeURIComponent(svg)));
export function renderContext(design, person = {}) {
  const random = seeded(design.seed),
    fields = [
      "full_name",
      "given_names",
      "family_name",
      "birth_date",
      "sex",
      "nationality",
      "address_line1",
      "address_line2",
      "document_number",
      "email",
      "phone_number",
      "issue_date",
      "expiry_date",
      "height",
      "weight",
      "eye_color",
      "blood_type",
      "issuing_authority",
      "document_type",
      "mrz",
    ];
  const identity = Object.fromEntries(fields.map((k) => [k, person[k] ?? "—"]));
  identity.address_line2 = person.address_line2 || "";
  identity.mrz = person.mrz || "";
  const payload = "SYNTHETIC-SPECIMEN-" + (Number(design.seed) >>> 0);
  const code = (bcid) =>
    svgURL(
      bwipjs.toSVG({
        bcid,
        text: payload,
        scale: 2,
        ...(bcid === "qrcode" ? {} : { height: 12 }),
        includetext: false,
      }),
    );
  const photo =
    person.images?.image_transparent ||
    person.images?.image ||
    person.photo_url ||
    svgURL(
      '<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400"><rect width="300" height="400" fill="#dce5e5"/><circle cx="150" cy="135" r="62" fill="#8b9c9c"/><path d="M35 365a115 135 0 0 1 230 0" fill="#8b9c9c"/><text x="150" y="390" text-anchor="middle" font-size="16">PORTRAIT PLACEHOLDER</text></svg>',
    );
  return {
    identity,
    common: design.common || {},
    assets: {
      photo,
      qrcode: code("qrcode"),
      barcode: code("code128"),
      slim_barcode: code("code128"),
      pdf417: code("pdf417"),
    },
    variation: {
      margin: 32 + Math.floor(random() * 9),
      font_size: 30 + Math.floor(random() * 5),
      label_size: 18,
      field_gap: 12,
      column_gap: 32,
      header_gap: 22,
      watermark_opacity: 0.1,
      dates_reversed: Math.floor(random() * 2),
      photo_background: "transparent",
    },
    card_font: { url: "", weight: 400 },
    person,
    missing: fields.filter((k) => !person[k]),
  };
}
export async function renderTemplate(design, person = {}) {
  if (design.template.length + design.css.length > 500000)
    throw Error("Template exceeds 500 KB");
  const context = renderContext(design, person);
  if (JSON.stringify(context).length > 12 * 1024 * 1024)
    throw Error("Render context exceeds 12 MB");
  const result = await new Promise((resolve, reject) => {
    const worker = new Worker(
      new URL("./template-worker.js", import.meta.url),
      { type: "module" },
    );
    const timeout = setTimeout(() => {
      worker.terminate();
      reject(Error("Template rendering timed out"));
    }, 3000);
    const finish = () => {
      clearTimeout(timeout);
      worker.terminate();
    };
    worker.onmessage = ({ data }) => {
      finish();
      data.error ? reject(Error(data.error)) : resolve(data);
    };
    worker.onerror = (event) => {
      finish();
      reject(Error(event.message));
    };
    worker.postMessage({
      design: {
        template: design.template,
        css: design.css,
        text_color: design.text_color,
      },
      context,
    });
  });
  return { ...result, person, context };
}
