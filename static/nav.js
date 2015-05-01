
function buildNavbar(domElem) {

  var navbar = document.createElement("div");
  navbar.className = 'livechan_navbar';

  var container = document.createElement("div");
  // channel name lable
  var channelLabel = document.createElement("span");
  channelLabel.className = 'livechan_channel_label';

  // mod indicator
  var mod = document.createElement("span");
  mod.className = 'livechan_mod_indicator_inactive';
  mod.textContent = "Anon";
  
  // usercounter
  var usercount = document.createElement("span");
  usercount.className = 'livechan_usercount';

  container.appendChild(mod);
  container.appendChild(channelLabel);
  container.appendChild(usercount);

  navbar.appendChild(container);
  
  domElem.appendChild(navbar);

  return {
    userCount: usercount,
    mod: mod,
  };
}
