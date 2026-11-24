// Public, credential-free renderer for local automation. Designs are supplied by
// the caller; this page never signs in or fetches private API resources.
import { renderCard } from "./render.js";
import { mapRow } from "./api.js";
import { cardPNG } from "./copy-image.js";
window.renderSpecimen = async (design, row) => {
  const result = await renderCard(design, {
    pixelRatio: 1,
    person: row
      ? mapRow(
          row,
          design.sample_source?.path,
          design.sample_source?.index,
          design.sample_source?.revision,
        )
      : {},
  });
  const blob = await cardPNG(
    result.html,
    design.width || 640,
    design.height || 400,
  );
  return {
    html: result.html,
    png: Array.from(new Uint8Array(await blob.arrayBuffer())),
  };
};
window.rendererReady = true;
