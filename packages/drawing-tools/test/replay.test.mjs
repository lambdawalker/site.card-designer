import test from "node:test";
import assert from "node:assert/strict";
test("seeded recipes survive serialization and preserve independent layer choices", async () => {
  const { parameters } =
    await import("../src/draw/card_background/parameters.ts");
  const a = parameters({
    family: "type-a",
    seed: 42,
    color: "#95b5c7",
    width: 640,
    height: 400,
  });
  assert.deepEqual(
    a,
    parameters({
      family: "type-a",
      seed: 42,
      color: "#95b5c7",
      width: 640,
      height: 400,
    }),
  );
  assert.deepEqual(a, JSON.parse(JSON.stringify(a)));
  const b = parameters({
    family: "type-b",
    seed: 42,
    color: "#95b5c7",
    width: 640,
    height: 400,
  });
  for (const role of ["gradient", "grid", "lines"]) {
    assert.deepEqual(
      a.layers.find((l) => l.role === role),
      b.layers.find((l) => l.role === role),
    );
  }
  assert.notDeepEqual(
    a,
    parameters({
      family: "type-a",
      seed: 43,
      color: "#95b5c7",
      width: 640,
      height: 400,
    }),
  );
  assert.throws(() =>
    parameters({
      family: "type-a",
      seed: 42,
      color: "#ffffff",
      width: 0,
      height: 400,
    }),
  );
});
