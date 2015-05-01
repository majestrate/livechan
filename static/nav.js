
function buildNavbar(domElem) {

  var navbar = document.createElement("div");
  navbar.className = 'livechan_navbar';

  // channel name lable
  var channelLabel = document.createElement("span");
  channelLabel.className = 'livechan_channel_label';

  // mod indicator
  var mod = document.createElement("span");
  mod.className = 'livechan_mod_indicator_inactive';
  mod.textContent = "User";
  
  // usercounter
  var usercount = document.createElement("span");
  usercount.className = 'livechan_usercount';

  navbar.appendChild(mod);
  navbar.appendChild(channelLabel);
  navbar.appendChild(usercount);

  
  domElem.appendChild(navbar);

  return {
    userCount: usercount,
    mod: mod,
  };
}
