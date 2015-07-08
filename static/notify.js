
/*
 * @brief Build Notification widget
 * @param domElem root element to put widget in
 */
function buildNotifyPane(domElem) {
  var pane = document.createElement("div");
  pane.className = "livechan_notify_pane";
  domElem.appendChild(pane);
  return pane;
}

/*
 * @brief Livechan Notification system
 * @param domElem root element to put Notification Pane in.
 */
function LivechanNotify(domElem) {
  this.pane = buildNotifyPane(domElem);
}

/* @brief inform the user with a message */
LivechanNotify.prototype.inform = function(str) {
  new Notify("livechan", {body: str}).show();
  /*
  //XXX: implement
  var elem = document.createElement("div");
  elem.className = "livechan_notify_node";
  elem.textContent = Date.now() + ": " + str;
  this.pane.appendChild(elem);
  this.rollover();
  */
}

LivechanNotify.protoype.onLivechanNotify = function(str) {
  this.inform(str);
}

LivechanNotify.protoype.onLivechanEvent = function(str) {
  this.inform(str);
}


/* @brief roll over old messages */
LivechanNotify.prototype.rollover = function() {
  while ( this.pane.childNodes.length > this.scrollback ) {
    this.pane.childNodes.removeChild(this.pane.childNodes[0]);
  }
}
