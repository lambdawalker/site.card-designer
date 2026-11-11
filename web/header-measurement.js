// Text/CSS are laid out at logical card dimensions before any background is drawn.
// Same-origin access is needed to measure; scripts remain disabled by the sandbox
// and the same CSP as the final card. The frame never receives focus or input.
export async function measureHeader(html,width,height) {
 const frame=document.createElement('iframe');
 frame.setAttribute('sandbox','allow-same-origin');frame.setAttribute('aria-hidden','true');frame.tabIndex=-1;
 frame.dataset.headerMeasurement='';
 frame.style.cssText='position:fixed;left:-10000px;top:0;border:0;visibility:hidden;pointer-events:none;width:'+width+'px;height:'+height+'px';
 let timeout;
 try{
  const loaded=new Promise((resolve,reject)=>{frame.onload=resolve;frame.onerror=()=>reject(Error('Could not lay out card header'))});
  frame.srcdoc=html;document.body.append(frame);
  return await Promise.race([
   (async()=>{
    await loaded;const doc=frame.contentDocument;
    await doc.fonts.ready;
    const root=doc.getElementById('view-port');
    const headers=[...root.querySelectorAll('[data-background-header],.card-header')];
    if(!headers.length)return null;
    await Promise.allSettled(headers.flatMap(header=>[...header.querySelectorAll('img')].map(img=>img.decode())));
    const origin=root.getBoundingClientRect(),rects=[];
    for(const header of headers){
     if(getComputedStyle(header).visibility!=='hidden')rects.push(header.getBoundingClientRect());
     // Include positioned descendants that can extend beyond the header box.
     const walker=doc.createTreeWalker(header,NodeFilter.SHOW_TEXT);
     while(walker.nextNode()){
      const node=walker.currentNode;
      if(!node.textContent.trim()||getComputedStyle(node.parentElement).visibility==='hidden')continue;
      const range=doc.createRange();range.selectNodeContents(node);rects.push(...range.getClientRects());
     }
     for(const img of header.querySelectorAll('img'))if(getComputedStyle(img).visibility!=='hidden')rects.push(...img.getClientRects());
    }
    const visible=rects.filter(rect=>rect.width>0&&rect.height>0);
    if(!visible.length)return null;
    return {left:Math.min(...visible.map(r=>r.left))-origin.left,top:Math.min(...visible.map(r=>r.top))-origin.top,right:Math.max(...visible.map(r=>r.right))-origin.left,bottom:Math.max(...visible.map(r=>r.bottom))-origin.top};
   })(),
   new Promise((_,reject)=>{timeout=setTimeout(()=>reject(Error('Timed out laying out card header')),10000)})
  ]);
 }finally{clearTimeout(timeout);frame.remove()}
}
