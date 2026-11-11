<script>
 import {onMount,tick} from 'svelte';
 import {downloadName,renderedPNG,saveBlob,zipFiles} from './download.js';
 import {copyCardImage} from './copy-image.js';
 import {formatOutput} from './format-output.js';
 import Editor from './Editor.svelte';

 let workspace='designer';
 import Icon from './Icon.svelte';
 import {api,renderCard,backgroundRecipe} from './render.js';
 import {starter} from './starter.js';
 import {themedDesign} from './color-theme.js';
 let design=themedDesign({...starter}), bundles=[], id=null, bundleName='', saved='', ready=false, busy=false;
 let copying=false,previewWidth=640,previewHeight=400;
 async function copyImage(){if(copying||!preview)return;copying=true;try{await copyCardImage(preview,previewWidth,previewHeight);notice='Card image copied'}catch(e){notice='Could not copy image: '+e.message}finally{copying=false}}
 let preview='',output='',renderedCSS='',context={},error='',notice='';
 let previewTabs=[],activePreviewId='',previewCounter=0;
 async function downloadCards(bundle=false){
   if(busy)return;
   rememberPreview();
   const snapshots=(bundle?previewTabs:[previewTabs.find(tab=>tab.key===activePreviewId)]).map(tab=>structuredClone(tab.design));
   await act(async()=>{
     const files=[];
     for(let index=0;index<snapshots.length;index++){
       const item=snapshots[index];notice='Rendering card '+(index+1)+' of '+snapshots.length+'…';
       const blob=await renderedPNG(item,renderCard);
       files.push({name:downloadName(item.template_id||item.name)+'-'+(index+1)+'.png',blob});
     }
     if(bundle)saveBlob(await zipFiles(files),downloadName(bundleName||'Card bundle')+'.zip');
     else saveBlob(files[0].blob,downloadName(snapshots[0].name)+'.png');
     notice=bundle?'Bundle downloaded':'Card downloaded';
   });
 }
 function previewState(){return {design:structuredClone(design),preview,output,renderedCSS,context:structuredClone(context),error,previewWidth,previewHeight}}
 function rememberPreview(){
   const state=previewState();
   if(!activePreviewId){activePreviewId='preview-'+(++previewCounter);previewTabs=[{key:activePreviewId,...state}]}
   else previewTabs=previewTabs.map(tab=>tab.key===activePreviewId?{...tab,...state}:tab);
 }
 function restorePreview(tab){
   serial++;randomSerial++;randomizing=false;
   activePreviewId=tab.key;design=structuredClone(tab.design);
   preview=tab.preview;output=tab.output;renderedCSS=tab.renderedCSS;context=structuredClone(tab.context);error=tab.error;
   previewWidth=tab.previewWidth;previewHeight=tab.previewHeight;notice='';
 }
 function switchPreview(key){if(key===activePreviewId||busy)return;rememberPreview();const tab=previewTabs.find(tab=>tab.key===key);if(tab)restorePreview(tab)}
 function previewTabKey(e,index){
   if(!['ArrowLeft','ArrowRight','Home','End'].includes(e.key))return;
   e.preventDefault();const next=e.key==='Home'?0:e.key==='End'?previewTabs.length-1:(index+(e.key==='ArrowRight'?1:-1)+previewTabs.length)%previewTabs.length;
   switchPreview(previewTabs[next].key);document.getElementById(previewTabs[next].key)?.focus();
 }
 function bundlePayload(name, tabs, active, current){
   return {name, layouts:tabs.map(tab=>tab.key===active?current:tab.design)};
 }
 function snapshotBundle(){
   return structuredClone(bundlePayload(bundleName,previewTabs,activePreviewId,design));
 }
 function layoutTab(layout){
   return {key:'preview-'+(++previewCounter),design:layout,preview:'',output:'',renderedCSS:'',
     context:{},error:'',previewWidth:layout.width||640,previewHeight:layout.height||400};
 }
 function installBundle(bundle){
   id=bundle?.id??null;
   bundleName=bundle?.name??'';
   const layouts=(bundle?.layouts??[{...starter}]).map(themedDesign);
   previewTabs=layouts.map(layoutTab);
   restorePreview(previewTabs[0]);
   saved=JSON.stringify(snapshotBundle());
 }
 async function openBundle(bundle){
   if(busy)return;
   await act(async()=>{
     if(!await canLeave())return;
     await open(bundle.id);
     closeTemplates();
   });
 }
 function addLayout(){
   rememberPreview();
   const tab=layoutTab(themedDesign({...starter,name:'Layout '+(previewTabs.length+1)}));
   previewTabs=[...previewTabs,tab];
   restorePreview(tab);
 }
 async function removeLayout(){
   if(previewTabs.length<2)return;
   if(!await askAction('delete','Remove layout','Remove “'+design.name+'” from this bundle?','Remove layout'))return;
   const index=previewTabs.findIndex(tab=>tab.key===activePreviewId);
   previewTabs=previewTabs.filter(tab=>tab.key!==activePreviewId);
   restorePreview(previewTabs[Math.min(index,previewTabs.length-1)]);
 }
 let theme=localStorage.getItem('studio-theme')||'system',zoom='fit',width=720,stageHeight=600,tab='HTML',serial=0;
 let media=window.matchMedia("(prefers-color-scheme: dark)"),systemDark=media.matches;
 $: dark=theme==="dark"||(theme==="system"&&systemDark);
 let recipeText='',recipeOpen=false,recipeError='',recipeWrap=true,fieldWrap=true;
 let randomizing=false,randomSerial=0;
 let settingsPanel,settingsTrigger,settingsOpen=false,settingsTop=110,settingsLeft=16;
 let datasetSettingsPanel,datasetSettingsOpen=false,datasetSettingsTop=110,datasetSettingsLeft=16;
 function toggleDatasetSettings(e){
   if(datasetSettingsPanel.matches(':popover-open')){datasetSettingsPanel.hidePopover();return}
   const rect=e.currentTarget.getBoundingClientRect();
   datasetSettingsTop=Math.max(12,Math.min(rect.bottom+8,window.innerHeight-150));
   datasetSettingsLeft=Math.max(12,Math.min(rect.left,window.innerWidth-292));
   datasetSettingsPanel.showPopover();
 }
 async function randomizeCard(){
   if(randomizing)return;
   const request=++randomSerial,source=sourceId,previous=design.sample;
   const families=['type-b','type-b-v1','type-c-v1','type-d-v1','type-a-v1','plain'].filter(family=>family!==design.background);
   const values=crypto.getRandomValues(new Uint32Array(2));
   design={...design,background:families[values[0]%families.length],seed:values[1],background_recipe:null};
   recipeOpen=false;
   if(source===null)return;
   randomizing=true;
   try{
     const record=await api('samples/random?source_id='+source+'&exclude='+encodeURIComponent(previous));
     if(request===randomSerial&&sourceId===source&&design.sample===previous)design={...design,sample:record.id,sample_source:record.sample_source};
   }catch(e){if(request===randomSerial)notice=e.message}
   finally{if(request===randomSerial)randomizing=false}
 }
 function positionSettings(){
   if(!settingsTrigger)return;
   const rect=settingsTrigger.getBoundingClientRect();
   settingsTop=Math.max(12,Math.min(rect.bottom+8,window.innerHeight-120));
   settingsLeft=Math.max(12,Math.min(rect.left,window.innerWidth-432));
 }
 function toggleSettings(){
   if(settingsPanel.matches(':popover-open'))settingsPanel.hidePopover();
   else{positionSettings();settingsPanel.showPopover()}
 }
 function editRecipe(){recipeText=JSON.stringify(context.recipe,null,2);recipeError='';recipeOpen=true}
 function applyRecipe(){
   try{design={...design,background_recipe:JSON.parse(recipeText)};recipeOpen=false}
   catch(e){recipeError=e.message}
 }
 let inspector='Inspect',datasetRefs=[],sourceError='',sourceId=null,sourceRecords=[],sourceSearch='',sourceQuery='',sourceOffset=0,sourceImage='',sourceColumns=4;
 let sourceListLoading=true,sourceLoading=false,sourceSerial=0,datasetsBooting=true,sourcePoll,sourcesDisposed=false;
 let sourceGrid,sourceMoreLoading=false,sourceHasMore=true;
 let datasetSplit,datasetGridSize=50,selectedField='';
 let selectedRecord=null,selectedFields={},selectedFieldTypes={},recordError='',recordLoading=false,recordSerial=0,recordKey;
 $: if(ready&&design.sample!==recordKey){recordKey=design.sample;loadRecord(design.sample)}
 $: recordInSource=selectedRecord?.source_id===sourceId&&selectedRecord!==null;
 $: recordFields=recordInSource?Object.entries(selectedFields):[];
 $: if(recordInSource&&!recordLoading&&!Object.hasOwn(selectedFields,selectedField))selectedField=recordFields[0]?.[0]??'';
 function fieldTabKey(e,index){
   if(!['ArrowLeft','ArrowRight','Home','End'].includes(e.key))return;
   e.preventDefault();
   const next=e.key==='Home'?0:e.key==='End'?recordFields.length-1:(index+(e.key==='ArrowRight'?1:-1)+recordFields.length)%recordFields.length;
   selectedField=recordFields[next][0];
   const target=e.currentTarget.parentElement.querySelectorAll('[role=tab]')[next];
   target.focus();target.scrollIntoView({block:'nearest',inline:'nearest'});
 }
 function resizeDataset(e){
   if(e.button!==0)return;
   const handle=e.currentTarget,bounds=datasetSplit.getBoundingClientRect();
   handle.setPointerCapture(e.pointerId);
   handle.onpointermove=event=>datasetGridSize=Math.max(15,Math.min(85,100*(event.clientY-bounds.top-4)/Math.max(1,bounds.height-8)));
   const release=()=>{handle.onpointermove=null;handle.onpointerup=null;handle.onpointercancel=null;handle.onlostpointercapture=null};
   handle.onpointerup=release;handle.onpointercancel=release;handle.onlostpointercapture=release;
 }
 function datasetResizeKey(e){
   if(!['ArrowUp','ArrowDown','Home','End'].includes(e.key))return;
   e.preventDefault();
   datasetGridSize=e.key==='Home'?15:e.key==='End'?85:Math.max(15,Math.min(85,datasetGridSize+(e.key==='ArrowDown'?5:-5)));
 }
 async function loadRecord(key){
   const current=++recordSerial;
   selectedRecord=null;selectedFields={};selectedFieldTypes={};recordError='';recordLoading=!!key;
   if(!key)return;
   try{const result=await api('samples/'+encodeURIComponent(key)+'?source='+encodeURIComponent(JSON.stringify(design.sample_source||null)));if(current===recordSerial){selectedRecord=result.record;selectedFields=result.fields;selectedFieldTypes=result.field_types}}
   catch(e){if(current===recordSerial)recordError=e.message}
   finally{if(current===recordSerial)recordLoading=false}
 }
 function fieldText(value){return typeof value==='string'?value:JSON.stringify(value,null,2)??String(value)}
 const sourceViews=new Map();
 $: activeSource=datasetRefs.find(source=>source.id===sourceId);
 $: sourceImageFields=activeSource?.image_fields??[...new Set(sourceRecords.flatMap(record=>Object.keys(record.images||{})))];
 function sourcePhoto(record,field){return field?record.images?.[field]:record.photo_url}
 $: if(design.auto_theme){
   const generated=themedDesign(design);
   if(generated.color!==design.color||generated.text_color!==design.text_color)design={...generated,background_recipe:null};
 }
 function generateColorTheme(){design={...design,auto_theme:true,seed:crypto.getRandomValues(new Uint32Array(1))[0],background_recipe:null}}
 
 async function loadSources(){
   if(sourcesDisposed)return;
   try{
     const status=await api('datasets/status');
     const sources=await api('datasets');
     if(sourcesDisposed)return;
     const wasLoading=datasetsBooting;
     datasetRefs=sources;datasetsBooting=status.loading;
     sourceListLoading=datasetsBooting&&!datasetRefs.length;
     if(sourceId===null)await selectSource(datasetRefs[0]?.id??null);
     else await browseSource();
     if(sourcesDisposed)return;
     if(status.error)sourceError=status.error;
     if(wasLoading&&!datasetsBooting){
       if(ready){schedule({...design});if(design.sample)loadRecord(design.sample)}
       if(templateDialog?.open)showTemplates();
     }
   }catch(e){sourceError=e.message;sourceListLoading=false;datasetsBooting=false}
   finally{if(!sourcesDisposed&&datasetsBooting)sourcePoll=setTimeout(loadSources,500)}
 }
 async function browseSource(start=0,append=false){
   const current=++sourceSerial,key=sourceId;
   if(!append)sourceQuery=sourceSearch;
   const query=sourceQuery;
   if(!append){sourceRecords=[];sourceOffset=0;sourceHasMore=true}
   sourceError='';sourceLoading=!append&&key!==null;sourceMoreLoading=append;
   if(key===null){sourceLoading=false;sourceMoreLoading=false;sourceHasMore=false;return}
   if(datasetRefs.find(source=>source.id===key)?.loading){sourceMoreLoading=false;return}
   try{
     const records=await api('samples?offset='+start+'&q='+encodeURIComponent(query)+'&source_id='+key);
     if(current===sourceSerial){sourceRecords=append?[...sourceRecords,...records]:records;sourceOffset=sourceRecords.length;sourceHasMore=records.length===24}
   }catch(e){if(current===sourceSerial)sourceError=e.message}
   finally{if(current===sourceSerial){sourceLoading=false;sourceMoreLoading=false;await tick();maybeMore()}}
 }
 function maybeMore(){
   if(sourceGrid&&sourceHasMore&&!sourceLoading&&!sourceMoreLoading&&!sourceError&&sourceGrid.clientHeight>0&&sourceGrid.scrollHeight-sourceGrid.scrollTop-sourceGrid.clientHeight<160)browseSource(sourceRecords.length,true);
 }
 $: if(sourceColumns&&datasetGridSize)tick().then(maybeMore);
 async function selectSource(key){
   if(datasetSettingsPanel?.matches(':popover-open'))datasetSettingsPanel.hidePopover();
   if(sourceId!==null)sourceViews.set(sourceId,{search:sourceSearch,query:sourceQuery,image:sourceImage,columns:sourceColumns,records:sourceRecords,hasMore:sourceHasMore,scroll:sourceGrid?.scrollTop??0});
   sourceSerial++;sourceLoading=false;sourceMoreLoading=false;sourceError='';sourceId=key;
   const view=sourceViews.get(key);
   sourceSearch=view?.search??'';sourceQuery=view?.query??'';sourceImage=view?.image??'';sourceColumns=view?.columns??4;
   if(view?.records.length){sourceRecords=view.records;sourceHasMore=view.hasMore;sourceOffset=sourceRecords.length;await tick();if(sourceGrid)sourceGrid.scrollTop=view.scroll;maybeMore()}
   else await browseSource();
 }
 onMount(()=>{loadSources();return ()=>{sourcesDisposed=true;clearTimeout(sourcePoll);sourceSerial++;recordSerial++}});
 let left=25,right=24,templateHeight=50,sourceEditors,templateEditor,cssEditor,assets=[];
 async function upload(e){
   const file=e.target.files[0];if(!file)return;
   if(!['image/png','image/jpeg','image/webp'].includes(file.type)||file.size>512*1024){notice='Choose a PNG, JPEG or WebP smaller than 512 KB';return}
   const data=await new Promise((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(reader.result);reader.onerror=reject;reader.readAsDataURL(file)});
   assets=[...assets.filter(a=>a.id!=='asset-inline'),{id:'asset-inline',name:file.name}];design={...design,background:'asset-inline',background_image:data};
 }
 $: document.documentElement.dataset.theme=theme;
 $: localStorage.setItem('studio-theme',theme);
 $: dirty=!!id && JSON.stringify(bundlePayload(bundleName,previewTabs,activePreviewId,design))!==saved;
 $: cardWidth=design.width||640;
 $: cardHeight=design.height||400;
 $: scale=zoom==='fit'?Math.min(1.6,Math.max(.05,Math.min((width-72)/cardWidth,(stageHeight-96)/cardHeight))):Number(zoom);
 async function refresh(){bundles=await api('bundles')}
 async function open(key){installBundle(key?await api('bundles/'+key):null)}
 let actionDialog,actionResolve=null,actionKind='',actionTitle='',actionMessage='',actionLabel='',actionValue='';
 function askAction(kind,title,message,label,value=''){
   if(actionResolve)return Promise.resolve(null);
   actionKind=kind;actionTitle=title;actionMessage=message;actionLabel=label;actionValue=value;
   return new Promise(resolve=>{
     actionResolve=resolve;
     tick().then(()=>{
       if(actionResolve!==resolve)return;
       actionDialog.showModal();
       if(kind==='name')actionDialog.querySelector('input')?.focus();
     });
   });
 }
 function finishAction(value=null){
   const resolve=actionResolve;actionResolve=null;
   if(actionDialog?.open)actionDialog.close();
   resolve?.(value);
 }
 async function canLeave(){
   return !dirty||await askAction('discard','Unsaved changes','This bundle has unsaved changes. Discard them and continue?','Discard changes')===true;
 }
 async function act(fn){if(busy)return;busy=true;try{await fn()}catch(e){notice=e.message}finally{busy=false}}
 onMount(()=>{
   const onTheme=e=>systemDark=e.matches;media.addEventListener("change",onTheme);
   act(async()=>{assets=await api("artwork");await refresh();await open(bundles[0]?.id);ready=true});
   return ()=>{media.removeEventListener("change",onTheme);serial++};
 });
 async function prepareSnapshot(snapshot){
   const layouts=[];
   for(const layout of snapshot.layouts){
     const resolved=themedDesign(layout);
     layouts.push({...resolved,background_recipe:await backgroundRecipe(resolved)});
   }
   return {...snapshot,layouts};
 }
 async function save(){
   if(!id||!bundleName.trim())return;
   const snapshot=snapshotBundle();
   const updated=await api('bundles/'+id,'PUT',await prepareSnapshot(snapshot));
   // Keep the server-assigned IDs when a newly added layout is saved again.
   previewTabs=previewTabs.map((tab,index)=>({
     ...tab,design:{...tab.design,layout_id:updated.layouts[index].layout_id}
   }));
   design={...design,layout_id:previewTabs.find(tab=>tab.key===activePreviewId).design.layout_id};
   snapshot.layouts.forEach((layout,index)=>layout.layout_id=updated.layouts[index].layout_id);
   saved=JSON.stringify(snapshot);
   await refresh();
   notice='Bundle saved';
 }
 let templateDialog,templateBundles=[],templates=[],templatesLoading=false,templatesError='',templateSerial=0;
 async function showTemplates(){
   if(!templateDialog.open)templateDialog.showModal();
   const current=++templateSerial;
   templates=[];templatesLoading=true;templatesError='';
   try{
     const available=await api('bundles');
     const loaded=await Promise.all(available.map(bundle=>api('bundles/'+bundle.id)));
     if(current!==templateSerial)return;
     templateBundles=loaded;
     const items=loaded.flatMap(bundle=>bundle.layouts.slice(0,3).map((layout,index)=>({
       id:bundle.id+'-'+index,bundle_id:bundle.id,layout,width:layout.width,height:layout.height,state:'loading'
     })));
     templates=items;
     templatesLoading=false;
     for(const item of items){
       await new Promise(resolve=>setTimeout(resolve,0));
       if(current!==templateSerial)return;
       try{
         const result=await renderCard(item.layout,{scale:Math.min(200/item.width,150/item.height),thumbnail:true});
         if(current!==templateSerial)return;
         templates=templates.map(entry=>entry.id===item.id?{...entry,html:result.html,state:'ready'}:entry);
       }catch{
         if(current===templateSerial)templates=templates.map(entry=>entry.id===item.id?{...entry,state:'error'}:entry);
       }
     }
   }catch(e){if(current===templateSerial)templatesError=e.message}
   finally{if(current===templateSerial)templatesLoading=false}
 }
 function closeTemplates(){templateSerial++;templateDialog.close()}
 async function newBundle(){
   if(!await canLeave())return;
   const name=await askAction('name','New bundle','Choose a name for your bundle.','Create bundle','Untitled bundle');
   if(!name)return;
   const bundle=await api('bundles','POST',{name,layouts:[{...starter,name:'Layout 1'}]});
   installBundle(bundle);
   await refresh();
 }
 async function clone(){
   if(!id||!bundleName.trim())return;
   const copy=await api('bundles/'+id+'/clone','POST',await prepareSnapshot(snapshotBundle()));
   installBundle(copy);
   await refresh();
 }
 async function remove(){
   if(await askAction('delete','Delete bundle','Delete “'+bundleName+'” and all its layouts?','Delete bundle')){
     await api('bundles/'+id,'DELETE');
     await refresh();
     await open(bundles[0]?.id);
   }
 }
 function schedule(d,displayScale=scale){
   const current=++serial;
   setTimeout(async()=>{if(current!==serial)return;
     try{const result=await renderCard(d,{scale:displayScale});if(current!==serial)return;
       preview=result.html;previewWidth=d.width||640;previewHeight=d.height||400;output=result.template;renderedCSS=result.css;context={...(result.context||{}),seed:d.seed,sample:result.person,background:d.background,recipe:result.background_recipe};error='';
       const formatted=await Promise.all([formatOutput(result.template,'html'),formatOutput(result.css,'css')]);
       if(current===serial){[output,renderedCSS]=formatted;rememberPreview()}
     }catch(e){if(current===serial){error=e.message;rememberPreview()}}
   },140)
 }
 $: if(ready&&id&&workspace==='designer')schedule({...design},scale);
 function key(e){
   if(workspace==='designer'&&(e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='s'){
     e.preventDefault();if(!actionDialog?.open)act(save);
   }
 }
 function resizeSource(e){
   const handle=e.currentTarget;
   const bounds=sourceEditors.getBoundingClientRect();
   handle.setPointerCapture(e.pointerId);
   handle.onpointermove=event=>templateHeight=Math.max(15,Math.min(85,100*(event.clientY-bounds.top)/bounds.height));
   const release=()=>{handle.onpointermove=null;handle.onpointerup=null;handle.onpointercancel=null};
   handle.onpointerup=release;handle.onpointercancel=release;
 }
 function sourceResizeKey(e){
   if(!['ArrowUp','ArrowDown','Home','End'].includes(e.key))return;
   e.preventDefault();
   templateHeight=e.key==='Home'?15:e.key==='End'?85:Math.max(15,Math.min(85,templateHeight+(e.key==='ArrowDown'?5:-5)));
 }
 function resize(e,side){
   const handle=e.currentTarget;handle.setPointerCapture(e.pointerId);
   handle.onpointermove=event=>{const percent=100*event.clientX/window.innerWidth;
     if(side==='left')left=Math.max(15,Math.min(38,percent));else right=Math.max(15,Math.min(38,100-percent))};
   handle.onpointerup=()=>{handle.onpointermove=null};
 }
</script>
<svelte:window on:resize={positionSettings} on:keydown={key} on:beforeunload={e=>{if(ready&&dirty){e.preventDefault();e.returnValue=''}}}/>
<header class="top"><strong><i>▧</i> Card Designer</strong><span class="tag">STUDIO</span>
<button class:active={workspace==='designer'} aria-pressed={workspace==='designer'} disabled={busy} on:click={()=>workspace='designer'}>Design</button>
{#if workspace==='designer'}
<button class="icon-button" title="New bundle" aria-label="New bundle" disabled={busy} on:click={()=>act(newBundle)}><Icon name="new"/></button>
<button class="icon-button open-templates" title="Open bundles" aria-label="Open bundles" disabled={busy} on:click={showTemplates}><Icon name="open"/></button>
<button class="icon-button" title="Clone bundle" aria-label="Clone bundle" disabled={busy||!id||!bundleName.trim()} on:click={()=>act(clone)}><Icon name="clone"/></button>
<button class="primary icon-button" title="Save bundle (Ctrl+S)" aria-label="Save bundle" disabled={busy||!id||!bundleName.trim()} on:click={()=>act(save)}><Icon name="save"/></button>
<button class="icon-button" title="Delete bundle" aria-label="Delete bundle" disabled={busy||!id} on:click={()=>act(remove)}><Icon name="delete"/></button>
{/if}<span class="grow"></span><label>Theme <select bind:value={theme}><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></label></header>
{#if workspace==='designer'}
{#if id}
<div class="toolbar"><label>Bundle <input bind:value={bundleName} maxlength="160" disabled={busy||!id}></label>
<label>Layout <input bind:value={design.name} maxlength="160" disabled={busy||!id}></label>
<button disabled={busy||!id} on:click={addLayout}>Add layout</button>
<button disabled={busy||!id||previewTabs.length<2} on:click={()=>act(removeLayout)}>Remove layout</button>
<button class="icon-button" bind:this={settingsTrigger} title="Background settings" aria-label="Background settings" aria-haspopup="dialog" aria-controls="background-settings" aria-expanded={settingsOpen} on:click={toggleSettings}><Icon name="background"/></button>
<button class="icon-button" title="Randomize card" aria-label="Randomize card" disabled={randomizing||busy} on:click={randomizeCard}><Icon name="shuffle"/></button>
</div>
<div id="background-settings" class="background-settings" popover="auto" role="dialog" aria-labelledby="background-settings-title" bind:this={settingsPanel} inert={busy} on:toggle={e=>settingsOpen=e.newState==='open'} style={'top:'+settingsTop+'px;left:'+settingsLeft+'px;max-height:calc(100dvh - '+(settingsTop+12)+'px)'}>
<div class="settings-heading"><h2 id="background-settings-title">Background settings</h2><button class="icon-button" title="Close background settings" aria-label="Close background settings" on:click={()=>settingsPanel.hidePopover()}><Icon name="close"/></button></div>
<div class="settings-controls">
<label for="background-family">Background</label><select id="background-family" bind:value={design.background}><option value="type-b">Type B · Original recipe</option><option value="type-b-v1">Type B · Gradients, grids, lines</option><option value="type-c-v1">Type C · With header</option><option value="type-d-v1">Type D · With country</option><option value="type-a-v1">Type A · Country and header</option><option value="plain">Solid</option>{#each assets as a}<option value={a.id}>{a.name}</option>{/each}</select>
<label class="import">＋ Artwork<input aria-label="Import artwork" type="file" accept="image/png,image/jpeg,image/webp" on:change={e=>act(()=>upload(e))}></label>
<label for="background-color">Color</label><input id="background-color" aria-label="Background color" type="color" bind:value={design.color} on:input={()=>{design.auto_theme=false;design.background_recipe=null}}>
<label class="theme-toggle"><input type="checkbox" bind:checked={design.auto_theme}> Generate theme from seed</label>
<button on:click={generateColorTheme}>Generate theme</button>
<label for="text-color">Text color</label><input id="text-color" aria-label="Text color" type="color" value={design.text_color||'#000000'} on:input={e=>{design={...design,text_color:e.target.value,auto_theme:false}}}>
<label for="background-seed">Seed</label><input id="background-seed" type="number" min="0" max="4294967295" bind:value={design.seed}>
<button disabled={!context.recipe} on:click={editRecipe}>Background parameters</button>
{#if recipeOpen}<div class="settings-recipe"><h3>Background recipe<button aria-label="Word wrap background recipe" aria-pressed={recipeWrap} on:click={()=>recipeWrap=!recipeWrap}>Wrap</button></h3><p>Seed, family and color must match the active background.</p>
<textarea wrap={recipeWrap?'soft':'off'} class:wrapped={recipeWrap} aria-label="Background recipe" bind:value={recipeText} spellcheck="false"></textarea>
{#if recipeError}<p role="alert">{recipeError}</p>{/if}
<div class="settings-recipe-actions"><button on:click={applyRecipe}>Apply</button><button on:click={()=>{design={...design,background_recipe:null};recipeOpen=false}}>Reset parameters</button><button on:click={()=>recipeOpen=false}>Cancel</button></div></div>{/if}
</div></div>
<main inert={busy} style={'--left:'+left+'%;--right:'+right+'%'}>
<section class="sources"><div class="bar">01 / SOURCE <small>EDITABLE</small></div>
<div class="source-editors" style={'--template-height:'+templateHeight+'%'} bind:this={sourceEditors}>
<div class="source-pane"><h3>◇ Template <span class="editor-actions"><em>HTML / Jinja</em><button title="Format template (Ctrl+Alt+L)" aria-label="Format template" on:click={()=>templateEditor?.format()}>Format</button></span></h3><Editor bind:this={templateEditor} {dark} value={design.template} label="Template source" change={v=>design.template=v}/></div>
<button class="grip row-grip" aria-label="Resize template and stylesheet panels" aria-valuenow={templateHeight} aria-valuemin="15" aria-valuemax="85" role="separator" aria-orientation="horizontal" on:pointerdown={resizeSource} on:keydown={sourceResizeKey}></button>
<div class="source-pane"><h3># Stylesheet <span class="editor-actions"><em>CSS / Jinja</em><button title="Format stylesheet (Ctrl+Alt+L)" aria-label="Format stylesheet" on:click={()=>cssEditor?.format()}>Format</button></span></h3><Editor bind:this={cssEditor} {dark} value={design.css} language="css" label="CSS source" change={v=>design.css=v}/></div>
</div></section>
<button class="grip" aria-label="Resize source panel" on:pointerdown={e=>resize(e,'left')} on:keydown={e=>{if(e.key==='ArrowRight')left=Math.min(38,left+1);if(e.key==='ArrowLeft')left=Math.max(15,left-1)}}></button>
<section class="canvas"><div class="bar">02 / PREVIEW <span class="editor-actions"><button class="icon-button" aria-label="Download card PNG" title="Download card PNG" disabled={busy||!preview} on:click={()=>downloadCards()}><Icon name="download"/></button>{#if id}<button class="icon-button" aria-label="Download bundle ZIP" title="Download bundle ZIP" disabled={busy} on:click={()=>downloadCards(true)}><Icon name="archive"/></button>{/if}<button class="icon-button" aria-label="Copy card image" title="Copy card image" disabled={copying||!preview} on:click={copyImage}><Icon name="clone"/></button><select aria-label="Zoom" bind:value={zoom}><option value="fit">Fit</option><option value=".5">50%</option><option value="1">100%</option><option value="1.5">150%</option></select></span></div>
{#if previewTabs.length>1}<nav class="preview-tabs" role="tablist" aria-label="Card layouts">{#each previewTabs as tab,index (tab.key)}<button id={tab.key} role="tab" aria-selected={activePreviewId===tab.key} aria-controls="card-preview-panel" tabindex={activePreviewId===tab.key?0:-1} disabled={busy} class:active={activePreviewId===tab.key} on:click={()=>switchPreview(tab.key)} on:keydown={e=>previewTabKey(e,index)}>{activePreviewId===tab.key?design.name:tab.design.name}</button>{/each}</nav>{/if}
<div class="preview-tab-panel" id="card-preview-panel" role={previewTabs.length>1?'tabpanel':undefined} aria-labelledby={previewTabs.length>1?activePreviewId:undefined}>
<div class="stage" bind:clientWidth={width} bind:clientHeight={stageHeight}><div class="art"><div class="ruler" aria-hidden="true" style={"background-size:"+scale*32+"px 100%"}></div><small>SPECIMEN　 /　{cardWidth} × {cardHeight}</small><div style={'width:'+cardWidth*scale+'px;height:'+cardHeight*scale+'px'}><iframe title="Card preview" sandbox="" srcdoc={preview} style={'width:'+cardWidth+'px;height:'+cardHeight+'px;transform:scale('+scale+')'}></iframe></div></div></div>
{#if error}<div role="alert">{error}</div>{/if}</div></section>
<button class="grip" aria-label="Resize inspector" on:pointerdown={e=>resize(e,'right')} on:keydown={e=>{if(e.key==='ArrowLeft')right=Math.min(38,right+1);if(e.key==='ArrowRight')right=Math.max(15,right-1)}}></button>
<section class="inspect" class:exploring={inspector==='Data sources'}><div class="bar">03 / EXPLORE</div>
<nav aria-label="Right panel">{#each ['Inspect','Data sources'] as panel}<button class:active={inspector===panel} on:click={()=>{inspector=panel;if(panel==='Data sources'&&!sourceListLoading&&sourceId!==null){if(!sourceRecords.length)browseSource();else tick().then(maybeMore)}}}>{panel}</button>{/each}</nav>
{#if inspector==='Data sources'}
<div class="dataset-tabs-bar">
<div class="dataset-tabs" role="tablist" aria-label="Dataset references" aria-busy={sourceListLoading}>
{#if sourceListLoading}
<button class="active" role="tab" aria-selected="true" aria-controls="sources-loading-panel" disabled><span class="source-spinner" aria-hidden="true"></span>Loading datasets…</button>
{/if}
{#each datasetRefs as source,index (source.id)}
<button id={'source-tab-'+source.id} role="tab" aria-selected={sourceId===source.id} aria-controls={'source-panel-'+source.id} class:active={sourceId===source.id} title="lambdaWalker/ds.photo-id" on:click={()=>selectSource(source.id)}>{#if sourceId===source.id&&sourceLoading}<span class="source-spinner" aria-hidden="true"></span>{/if}Synthetic portraits · Hugging Face</button>
{/each}
</div>
</div>
<div class="dataset-explorer" id={activeSource?'source-panel-'+activeSource.id:undefined} role={activeSource?'tabpanel':undefined} aria-labelledby={activeSource?'source-tab-'+activeSource.id:undefined} aria-busy={sourceListLoading||sourceLoading}>
<div class="dataset-controls">
{#if activeSource}

<div id="dataset-settings" class="background-settings dataset-settings" popover="auto" role="dialog" aria-labelledby="dataset-settings-title" bind:this={datasetSettingsPanel} on:toggle={e=>datasetSettingsOpen=e.newState==='open'} style={'top:'+datasetSettingsTop+'px;left:'+datasetSettingsLeft+'px;max-height:calc(100dvh - '+(datasetSettingsTop+12)+'px)'}>
<div class="settings-heading"><h2 id="dataset-settings-title">Dataset settings</h2><button class="icon-button" title="Close dataset settings" aria-label="Close dataset settings" on:click={()=>datasetSettingsPanel.hidePopover()}><Icon name="close"/></button></div>
<p>lambdaWalker/ds.photo-id · immutable package cache</p>
<div class="source-display-controls"><label>Display image<select aria-label="Display image" bind:value={sourceImage}><option value="">Default image</option>{#each sourceImageFields as field}<option value={field}>{field}</option>{/each}</select></label><label>Columns<select aria-label="Dataset columns" bind:value={sourceColumns}>{#each Array.from({length:8},(_,i)=>i+1) as columns}<option value={columns}>{columns}</option>{/each}</select></label></div>
</div>
{#if activeSource.error}<p role="alert">{activeSource.error}</p>{/if}
<form class="source-search" on:submit|preventDefault={()=>browseSource()}><button type="button" class="icon-button" title="Dataset settings" aria-label="Dataset settings" aria-haspopup="dialog" aria-controls="dataset-settings" aria-expanded={datasetSettingsOpen} on:click={toggleDatasetSettings}><Icon name="settings"/></button><input aria-label="Search source records" placeholder="Search loaded packages…" bind:value={sourceSearch}><button>Search</button></form><p class="dataset-count">Search covers loaded packages. Clear search and scroll to load more portraits.</p>

{/if}
{#if sourceError}<p role="alert">{sourceError}</p>{/if}
</div>
<div class="dataset-split" bind:this={datasetSplit} style={'grid-template-rows:minmax(0,'+datasetGridSize+'fr) 8px minmax(0,'+(100-datasetGridSize)+'fr)'}>
<div class="dataset-grid-pane" aria-label="Dataset records">
{#if sourceListLoading}<p id="sources-loading-panel" role="status"><span class="source-spinner" aria-hidden="true"></span> Loading datasets…</p>
{:else if !activeSource}<p>Sign in to browse portraits.</p>
{:else}
{#if sourceLoading}<p role="status"><span class="source-spinner" aria-hidden="true"></span> Loading records…</p>
{:else}
<div class="records" bind:this={sourceGrid} on:scroll={maybeMore} style={'grid-template-columns:repeat('+sourceColumns+',minmax(0,1fr))'}>{#each sourceRecords as record}<button class:selected={design.sample===record.id} aria-pressed={design.sample===record.id} on:click={()=>design={...design,sample:record.id,sample_source:record.sample_source}}>{#if design.sample===record.id}<span class="selection-mark" aria-hidden="true">Selected</span>{/if}{#if sourcePhoto(record,sourceImage)}<img src={sourcePhoto(record,sourceImage)} alt="">{:else}<span class="missing-thumbnail">{sourceImage?'No '+sourceImage:'No image'}</span>{/if}{record.name}</button>{/each}</div>
{#if !sourceRecords.length}<p>No matching records.</p>{/if}
{/if}
<div class="source-scroll-status">{#if sourceMoreLoading}<span role="status"><span class="source-spinner" aria-hidden="true"></span>Loading more records…</span>{:else if sourceError}<button on:click={()=>browseSource(sourceRecords.length,true)}>Retry loading records</button>{:else if !sourceHasMore&&sourceRecords.length}<small>All matching records loaded</small>{/if}</div>
{/if}
</div>
<button class="grip row-grip dataset-divider" aria-label="Resize dataset grid and record fields" role="separator" aria-orientation="horizontal" aria-valuenow={Math.round(datasetGridSize)} aria-valuemin="15" aria-valuemax="85" on:pointerdown={resizeDataset} on:keydown={datasetResizeKey}></button>
<section class="dataset-fields-pane" aria-labelledby="record-fields-title" aria-busy={recordLoading}>
<h3 id="record-fields-title">Record fields {#if recordInSource}<small>{selectedRecord.name}</small>{#if selectedFieldTypes[selectedField]!=='rawImage'}<button aria-label="Word wrap record field" aria-pressed={fieldWrap} on:click={()=>fieldWrap=!fieldWrap}>Wrap</button>{/if}{/if}</h3>
{#if recordInSource&&!recordLoading}
<nav class="record-field-tabs" role="tablist" aria-label="Record fields">{#each recordFields as [field,value],index}<button id={'record-field-tab-'+index} role="tab" aria-selected={selectedField===field} aria-controls="record-field-panel" tabindex={selectedField===field?0:-1} class:active={selectedField===field} on:click={()=>selectedField=field} on:keydown={e=>fieldTabKey(e,index)}>{field}</button>{/each}</nav>
{/if}
<div class="record-fields-scroll">
{#if recordLoading||(datasetsBooting&&design.sample)}<p role="status"><span class="source-spinner" aria-hidden="true"></span> Loading fields…</p>
{:else if recordError}<p role="alert">{recordError}</p>
{:else if recordInSource&&recordFields.length}
<div class="record-field-panel" class:word-wrapped={fieldWrap} id="record-field-panel" role="tabpanel" tabindex="0" aria-labelledby={'record-field-tab-'+recordFields.findIndex(([field])=>field===selectedField)}>
{#if selectedFieldTypes[selectedField]==='rawImage'}<img src={selectedFields[selectedField]} alt={selectedField+' for '+selectedRecord.name}>{:else}<pre>{fieldText(selectedFields[selectedField])}</pre>{/if}
</div>
{:else if recordInSource}<p>No fields available for this record.</p>
{:else}<p>Select a record in this dataset to inspect its fields.</p>{/if}
</div>
</section>
</div>
</div>
{:else}
<nav>{#each ['HTML','CSS','Context'] as t}<button class:active={tab===t} on:click={()=>tab=t}>{t}</button>{/each}</nav>
{#key tab}
  <Editor readOnly {dark} label={`Rendered ${tab}`}
    language={tab==='Context'?'json':tab.toLowerCase()}
    value={tab==='HTML'?output:tab==='CSS'?renderedCSS:JSON.stringify(context,null,2)} />
{/key}{/if}</section></main>
{:else}<p class="gallery-status">Create a bundle or open one from the gallery.</p>{/if}
<footer>{notice|| (dirty?'● Modified':id?.startsWith('stock:')?'● Built-in template':id?'● Saved':'No bundle open')} <span>Bundle {id??"—"}　 /　 Seed {design.seed}</span></footer>

<dialog class="template-dialog" bind:this={templateDialog} aria-labelledby="template-dialog-title" aria-describedby="template-dialog-help" on:cancel={e=>{if(busy){e.preventDefault();return}templateSerial++}}>
<div class="template-dialog-header"><div><h2 id="template-dialog-title">Open a bundle</h2><p id="template-dialog-help">Choose a bundle to open its layouts in preview tabs.</p></div><button class="icon-button" title="Close bundles" aria-label="Close bundles" disabled={busy} on:click={closeTemplates}><Icon name="close"/></button></div>
{#if templatesLoading}<p class="gallery-status" role="status"><span class="source-spinner" aria-hidden="true"></span> Loading bundles…</p>
{:else if templatesError}<div class="gallery-status" role="alert"><p>{templatesError}</p><button on:click={showTemplates}>Try again</button></div>
{:else if !templates.length}<p class="gallery-status">No bundles available.</p>
{:else}<div class="bundle-gallery">{#each templateBundles as bundle (bundle.id)}
<button class="bundle-choice" disabled={busy} on:click={()=>openBundle(bundle)} aria-label={'Open bundle '+bundle.name} aria-describedby={'bundle-count-'+bundle.id}>
<span class="bundle-stack" aria-hidden="true">
{#each templates.filter(item=>item.bundle_id===bundle.id).slice(0,3) as item,index (item.id)}
<span class="stack-card" style={'width:'+item.width*Math.min(200/item.width,150/item.height)+'px;height:'+item.height*Math.min(200/item.width,150/item.height)+'px;left:calc(50% - '+(item.width*Math.min(200/item.width,150/item.height)/2+(Math.min(3,bundle.layouts.length)-1)*9)+'px + '+index*18+'px);top:'+((200-item.height*Math.min(200/item.width,150/item.height)-(Math.min(3,bundle.layouts.length)-1)*10)/2+index*10)+'px;z-index:'+(index+1)+';transform:rotate('+((index-1)*4)+'deg)'}>
{#if item.state==='ready'}<iframe title={'Bundle preview '+item.id} sandbox="" tabindex="-1" srcdoc={item.html} style={'width:'+item.width+'px;height:'+item.height+'px;transform:scale('+Math.min(200/item.width,150/item.height)+')'}></iframe>{:else if item.state==='loading'}<span class="source-spinner"></span>{:else}<span>Preview unavailable</span>{/if}
</span>{/each}
</span>
<span class="bundle-copy"><strong>{bundle.name}</strong><span class="bundle-count" id={'bundle-count-'+bundle.id}>{bundle.layouts.length} {bundle.layouts.length===1?'layout':'layouts'}</span></span>
</button>{/each}</div>{/if}
</dialog>

<dialog class="action-dialog" bind:this={actionDialog} aria-labelledby="action-title" aria-describedby="action-message" on:cancel={e=>{e.preventDefault();finishAction()}}>
<form on:submit|preventDefault={()=>finishAction(actionKind==='name'?actionValue.trim():true)}>
<h2 id="action-title">{actionTitle}</h2><p id="action-message">{actionMessage}</p>
{#if actionKind==='name'}<label for="new-bundle-name">Bundle name</label><input id="new-bundle-name" bind:value={actionValue} required maxlength="160">{/if}
<div class="dialog-actions"><button type="button" on:click={()=>finishAction()}>{actionKind==='discard'?'Keep editing':'Cancel'}</button><button class:danger={actionKind!=='name'} class:primary={actionKind==='name'} disabled={actionKind==='name'&&!actionValue.trim()}>{actionLabel}</button></div>
</form></dialog>

{/if}