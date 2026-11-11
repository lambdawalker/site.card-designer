let accessToken = "",
  expires = 0,
  config;
const transactionKey = "card-designer-oauth";
export const authenticated = () => !!accessToken && Date.now() < expires;
export const bearer = () => (authenticated() ? accessToken : "");
const base64url = (bytes) =>
  btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
export async function getConfig() {
  return (config ??= await (await fetch("/api/config")).json());
}
function domain(c) {
  return c.cognito_domain.startsWith("https://")
    ? c.cognito_domain
    : "https://" + c.cognito_domain;
}
export async function signIn() {
  const c = await getConfig();
  const verifier = base64url(crypto.getRandomValues(new Uint8Array(48))),
    state = base64url(crypto.getRandomValues(new Uint8Array(32)));
  sessionStorage.setItem(
    transactionKey,
    JSON.stringify({ verifier, state, created: Date.now() }),
  );
  const challenge = base64url(
    new Uint8Array(
      await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier)),
    ),
  );
  location.assign(
    domain(c) +
      "/oauth2/authorize?" +
      new URLSearchParams({
        client_id: c.client_id,
        response_type: "code",
        scope: "openid email profile",
        redirect_uri: c.redirect_uri,
        state,
        code_challenge: challenge,
        code_challenge_method: "S256",
      }),
  );
}
export function validTransaction(tx, state, now = Date.now()) {
  return (
    !!tx &&
    typeof state === "string" &&
    state.length > 20 &&
    state === tx.state &&
    typeof tx.verifier === "string" &&
    now - tx.created >= 0 &&
    now - tx.created < 600000
  );
}
export async function finishSignIn() {
  const params = new URLSearchParams(location.search);
  if (!params.has("code") && !params.has("error")) return;
  const raw = sessionStorage.getItem(transactionKey);
  sessionStorage.removeItem(transactionKey);
  history.replaceState({}, "", location.pathname);
  const tx = raw ? JSON.parse(raw) : null;
  if (!validTransaction(tx, params.get("state")))
    throw Error("Invalid or expired sign-in state. Please sign in again.");
  if (params.has("error"))
    throw Error(params.get("error_description") || params.get("error"));
  const c = await getConfig();
  const res = await fetch(domain(c) + "/oauth2/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      client_id: c.client_id,
      redirect_uri: c.redirect_uri,
      code: params.get("code"),
      code_verifier: tx.verifier,
    }),
  });
  const data = await res.json();
  if (!res.ok || !data.access_token)
    throw Error("Sign-in token exchange failed");
  accessToken = data.access_token;
  expires = Date.now() + Number(data.expires_in || 3600) * 1000;
}
export async function signOut() {
  accessToken = "";
  expires = 0;
  sessionStorage.removeItem(transactionKey);
  const c = await getConfig();
  location.assign(
    domain(c) +
      "/logout?" +
      new URLSearchParams({ client_id: c.client_id, logout_uri: c.logout_uri }),
  );
}
