// Browser port of one Type B recipe: linear gradient, rotated dot grid, parallel sine waves.
function typeBBackground(width, height, color, seed, target = {width, height}) {
  let state = seed >>> 0;
  function random() {
    state = (Math.imul(1664525, state) + 1013904223) >>> 0;
    return state / 4294967296;
  }
  const canvas = document.createElement('canvas');
  canvas.width = target.width;
  canvas.height = target.height;
  const ctx = canvas.getContext('2d');
  ctx.scale(target.width / width, target.height / height);
  const gradient = ctx.createLinearGradient(0, random() * height, width, random() * height);
  gradient.addColorStop(0, '#fff');
  gradient.addColorStop(1, color);
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, width, height);

  const extent = Math.ceil(Math.hypot(width, height));
  ctx.translate(width / 2, height / 2);
  ctx.rotate(random() * Math.PI);
  ctx.fillStyle = '#193c4a';
  ctx.globalAlpha = 0.23;
  const radius = 1 + random() * 2;
  const step = radius * 2 + 6 + random() * 8;
  ctx.beginPath();
  for (let y = -extent; y <= extent; y += step) {
    for (let x = -extent; x <= extent; x += step) {
      ctx.moveTo(x + radius, y);
      ctx.arc(x, y, radius, 0, 2 * Math.PI);
    }
  }
  ctx.fill();
  ctx.rotate(random() * Math.PI);
  ctx.strokeStyle = '#204f57';
  const amplitude = 12 + random() * 24;
  const spacing = 8 + random() * 12;
  const frequency = 0.008 + random() * 0.015;
  ctx.beginPath();
  for (let y = -extent; y <= extent; y += spacing) {
    ctx.moveTo(-extent, y + amplitude * Math.sin(-extent * frequency));
    for (let x = -extent + 2; x <= extent; x += 2) {
      ctx.lineTo(x, y + amplitude * Math.sin(x * frequency));
    }
  }
  ctx.stroke();
  return canvas.toDataURL();
}
