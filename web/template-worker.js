import nunjucks from "nunjucks/browser/nunjucks.js";
const environment = new nunjucks.Environment(null, {
  autoescape: true,
  throwOnUndefined: false,
});
// This realm never receives credentials. Deny network access even for malicious
// template expressions that discover JavaScript globals.
self.fetch = () =>
  Promise.reject(Error("Network unavailable in template renderer"));
self.XMLHttpRequest = undefined;
self.WebSocket = undefined;
self.importScripts = () => {
  throw Error("Imports unavailable");
};
self.onmessage = ({ data }) => {
  try {
    const { design, context } = data;
    context.env = {
      theme: {
        text_color: { to_rgb_hex: () => design.text_color || "#000000" },
      },
    };
    self.postMessage({
      template: environment.renderString(design.template, context),
      css: environment.renderString(design.css, context),
    });
  } catch (error) {
    self.postMessage({ error: error.message });
  }
};
