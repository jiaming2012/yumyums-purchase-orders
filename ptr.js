// Pull-to-refresh for iOS standalone PWA (Android handles it natively)
(function(){
  if(!navigator.standalone&&!window.matchMedia('(display-mode:standalone)').matches)return;
  var startY=0,pulling=false,indicator=null;
  function getIndicator(){
    if(!indicator){
      indicator=document.createElement('div');
      indicator.style.cssText='position:fixed;top:0;left:0;right:0;height:0;display:flex;align-items:center;justify-content:center;font-size:13px;color:#999;overflow:hidden;transition:height .2s;z-index:9999;background:inherit';
      indicator.textContent='Release to refresh';
      document.body.prepend(indicator);
    }
    return indicator;
  }
  document.addEventListener('touchstart',function(e){
    if(window.scrollY===0&&e.touches.length===1){
      startY=e.touches[0].pageY;
      pulling=true;
    }
  },{passive:true});
  document.addEventListener('touchmove',function(e){
    if(!pulling)return;
    var dy=e.touches[0].pageY-startY;
    if(dy>0&&dy<150){
      var ind=getIndicator();
      ind.style.height=Math.min(dy*0.5,50)+'px';
      ind.textContent=dy>80?'Release to refresh':'Pull to refresh';
    }
  },{passive:true});
  document.addEventListener('touchend',function(e){
    if(!pulling)return;
    pulling=false;
    var ind=getIndicator();
    var h=parseInt(ind.style.height);
    if(h>=40){
      ind.textContent='Refreshing\u2026';
      setTimeout(function(){location.reload()},200);
    }else{
      ind.style.height='0';
    }
  },{passive:true});
})();
