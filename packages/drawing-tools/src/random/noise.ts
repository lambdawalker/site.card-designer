const fade = (t: number) => t * t * t * (t * (t * 6 - 15) + 10);
export function noise(x: number, y: number, seed: number): number {
  const gradient = (ix: number, iy: number, dx: number, dy: number) => {
    let h = Math.imul(ix ^ seed, 374761393) ^ Math.imul(iy, 668265263);
    h = Math.imul(h ^ (h >>> 13), 1274126177);
    const a = ((h >>> 0) / 4294967296) * Math.PI * 2;
    return Math.cos(a) * dx + Math.sin(a) * dy;
  };
  const ix = Math.floor(x),
    iy = Math.floor(y),
    dx = x - ix,
    dy = y - iy,
    u = fade(dx),
    v = fade(dy);
  const a =
    gradient(ix, iy, dx, dy) * (1 - u) + gradient(ix + 1, iy, dx - 1, dy) * u;
  const b =
    gradient(ix, iy + 1, dx, dy - 1) * (1 - u) +
    gradient(ix + 1, iy + 1, dx - 1, dy - 1) * u;
  return a * (1 - v) + b * v;
}
