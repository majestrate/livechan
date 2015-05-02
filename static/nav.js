

/* 
 *  @brief build livechan navbar
 *  @param domElem the root element to put the navbar in
 */
function LivechanNavbar(domElem) {

  this.navbar = document.createElement("div");
  this.navbar.className = 'livechan_navbar';

  var container = document.createElement("div");
  // channel name label
  var channelLabel = document.createElement("span");
  channelLabel.className = 'livechan_navbar_channel_label';

  this.channel = channelLabel;
  
  // mod indicator
  this.mod = document.createElement("span");
  this.mod.className = 'livechan_navbar_mod_indicator_inactive';

  // TODO: don't hardcode
  this.mod.textContent = "Anon";

  // usercounter
  this.status = document.createElement("span");
  this.status.className = 'livechan_navbar_status';

  container.appendChild(this.mod);
  container.appendChild(this.channel);
  container.appendChild(this.status);

  navbar.appendChild(container);
  
  domElem.appendChild(this.navbar);

}


/* @brief called when there is an "event" for the navbar */
LivechanNavbar.prototype.onLivechanEvent = function (evstr) {
  if ( evstr === "login:mod" ) {
    // set indicator
    this.mod.className = "livechan_mod_indicator_active";
    this.mod.textContent = "Moderator";
  } else if ( evstr === "login:admin" ) {
    this.mod.className = "livechan_mod_indicator_admin";
    this.mod.textContent = "Admin";
  }
}

/* @brief called when there is a notification for us */
LivechanNavbar.prototype.onLivechanNotify = function(evstr) {
  // do nothing for now
  // maybe have some indicator that shows number of messages unread?
}

/* @brief update online user counter */
LivechanNavbar.prototype.updateUsers = function(count) {
  this.updateStatus("Online: "+count);
}

/* @brief update status label */
LivechanNavbar.prototype.updateStatus = function(str) {
  this.status.textContent = str;
}

/* @brief set channel name */
LivechanNavbar.prototype.setChannel = function(str) {
  this.channel.textContent = str;
}
