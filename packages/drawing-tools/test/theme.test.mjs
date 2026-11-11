import test from "node:test";
import assert from "node:assert/strict";
import { generateTheme } from "../src/draw/color/theme.ts";
import { parameters } from "../src/draw/card_background/parameters.ts";

function luminance(hex) {
  const channels = [1, 3, 5].map((offset) => {
    const channel = parseInt(hex.slice(offset, offset + 2), 16) / 255;
    return channel <= 0.04045
      ? channel / 12.92
      : ((channel + 0.055) / 1.055) ** 2.4;
  });
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}

test("themes are deterministic and use a separate random stream", () => {
  const options = {
    family: "type-a", seed: 42, color: "#95b5c7", width: 640, height: 400,
  };
  const before = parameters(options);
  const theme = generateTheme(42);
  assert.deepEqual(theme, generateTheme(42));
  assert.notDeepEqual(theme, generateTheme(43));
  assert.deepEqual(before, parameters(options));
  assert.deepEqual(theme, JSON.parse(JSON.stringify(theme)));
});

test("themes preserve HSLuv ranges and accessible black text", () => {
  for (let seed = 0; seed < 10000; seed++) {
    const theme = generateTheme(seed);
    assert.match(theme.primary_color, /^#[0-9a-f]{6}$/);
    assert.ok(Number.isInteger(theme.hue) && theme.hue >= 0 && theme.hue <= 360);
    assert.ok(Number.isInteger(theme.saturation) && theme.saturation >= 30 && theme.saturation <= 60);
    assert.ok(Number.isInteger(theme.lightness) && theme.lightness >= 30 && theme.lightness <= 80);
    assert.equal(theme.text_color, "#000000");
    const background = luminance(theme.primary_color);
    const text = luminance(theme.text_color);
    const contrast = (Math.max(background, text) + 0.05) / (Math.min(background, text) + 0.05);
    assert.ok(contrast >= 4.5, `seed ${seed}: contrast ${contrast}`);
  }
});

test("themes do not depend on global Math.random", () => {
  const original = Math.random;
  Math.random = () => { throw new Error("Unexpected global randomness"); };
  try {
    assert.deepEqual(generateTheme(123), generateTheme(123));
  } finally {
    Math.random = original;
  }
});
