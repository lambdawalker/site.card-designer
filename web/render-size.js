// Logical card coordinates stay fixed; only the bitmap's backing resolution changes.
export function renderSize(design, {scale=1,pixelRatio=1,thumbnail=false}={}) {
  const width=design.width??640,height=design.height??400;
  if(!Number.isInteger(width)||!Number.isInteger(height)||width<1||height<1||width>4096||height>4096)throw Error('Invalid card dimensions');
  if(!Number.isFinite(scale)||scale<=0||!Number.isFinite(pixelRatio)||pixelRatio<=0)throw Error('Invalid render scale');
  const density=Math.min(Math.max(thumbnail?1/4096:1,scale*pixelRatio),4096/width,4096/height);
  return {width:Math.min(4096,Math.ceil(width*density)),height:Math.min(4096,Math.ceil(height*density))};
}
