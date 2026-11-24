async function dataURL(url) {
 const response=await fetch(url);
 if(!response.ok)throw Error('Could not load card image: '+response.status);
 const blob=await response.blob();
 return new Promise((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(reader.result);reader.onerror=()=>reject(Error('Could not read card image'));reader.readAsDataURL(blob)});
}
export async function cardPNG(html,width,height) {
 const doc=new DOMParser().parseFromString(html,'text/html');
 doc.querySelectorAll('script,iframe,meta,link,object,embed,base,foreignObject').forEach(node=>node.remove());
 for(const node of doc.querySelectorAll('*'))for(const attr of [...node.attributes])if(attr.name.toLowerCase().startsWith('on'))node.removeAttribute(attr.name);
 const images=new Map();
 const inline=url=>{if(url.startsWith('data:'))return Promise.resolve(url);throw Error('Card images must be embedded data URLs');const absolute=new URL(url,location.origin).href;if(!images.has(absolute))images.set(absolute,dataURL(absolute));return images.get(absolute)};
 for(const image of doc.querySelectorAll('img')){image.removeAttribute('srcset');if(image.getAttribute('src'))image.setAttribute('src',await inline(image.getAttribute('src')))}
 for(const node of [...doc.querySelectorAll('style'),...doc.querySelectorAll('[style]')]){
  const css=node.tagName==='STYLE'?node.textContent:node.getAttribute('style');
  let result=css;
  for(const match of css.matchAll(/url\(\s*(['"]?)(.*?)\1\s*\)/g)){if(match[2]&&!match[2].startsWith('data:')&&!match[2].startsWith('#'))result=result.replace(match[0],'url("'+await inline(match[2])+'")')}
  if(node.tagName==='STYLE')node.textContent=result;else node.setAttribute('style',result);
 }
 doc.documentElement.setAttribute('xmlns','http://www.w3.org/1999/xhtml');
 const content=new XMLSerializer().serializeToString(doc.documentElement);
 const svg='<svg xmlns="http://www.w3.org/2000/svg" width="'+width+'" height="'+height+'"><foreignObject width="100%" height="100%">'+content+'</foreignObject></svg>';
 const image=new Image();image.src='data:image/svg+xml;charset=utf-8,'+encodeURIComponent(svg);await image.decode();
 const canvas=document.createElement('canvas');canvas.width=width;canvas.height=height;canvas.getContext('2d').drawImage(image,0,0,width,height);
 return new Promise((resolve,reject)=>canvas.toBlob(blob=>blob?resolve(blob):reject(Error('Could not render card image')),'image/png'));
}
export async function copyCardImage(html,width,height) {
 if(!window.isSecureContext||!navigator.clipboard?.write||typeof ClipboardItem==='undefined')throw Error('Copy image requires HTTPS or localhost and a browser with image clipboard support.');
 // Supply a promise immediately so clipboard permission stays tied to the button click.
 await navigator.clipboard.write([new ClipboardItem({'image/png':cardPNG(html,width,height)})]);
}
