

function buildCaptcha(domElem, prefix) {
    var captcha_widget = document.createElement("div");
    captcha_widget.className = "livechan_captcha";

    var text = document.createElement("div");
    text.textContent = "solve the captcha";
    captcha_widget.appendChild(text);

    var captcha_image = document.createElement("img");
    captcha_image.className = "livechan_captcha_image";
    var div = document.createElement("div");
    div.appendChild(captcha_image);
    captcha_widget.appendChild(div);
    
    var captcha_entry = document.createElement("input");
    captcha_entry.className = "livechan_captcha_input";
    var div = document.createElement("div");
    div.appendChild(captcha_entry);
    captcha_widget.appendChild(div);

    var captcha_submit = document.createElement("input");
    captcha_submit.setAttribute("type", "button");
    captcha_submit.value = "solve";
    var div = document.createElement("div");
    div.appendChild(captcha_submit);
    captcha_widget.appendChild(div);

    domElem.appendChild(captcha_widget);
    
    return {
      widget: captcha_widget,
      button: captcha_submit,
      image: captcha_image,
      entry: captcha_entry,
      prefix: prefix,
    }
}

function Captcha(domElem, options) {
  if (options) {
    this.options = options;
  } else {
    this.options = {};
  }
  
  this.prefix = options.prefix || "/";  
  this.widget = buildCaptcha(domElem, this.prefix);
  var self = this;
  this.widget.button.addEventListener("click", function() { self.process(); });
}

Captcha.prototype.load = function() {
  var self = this;
  var xhr = new XMLHttpRequest();

  // TODO: https detection
  var url = "http://" + location.hostname + this.prefix ;

  xhr.open('get', url +"/captcha.json");
  xhr.onreadystatechange = function () {
    if (xhr.readyState == 4 && xhr.status == 200) {
      var jdata = JSON.parse(xhr.responseText);
      if ( jdata.captcha ) {
        self.setCaptchaId(jdata.captcha);
      }
    }
  }
    
  xhr.send();
}

/**
 * @brief set captcha id
 */
Captcha.prototype.setCaptchaId = function(id) {
  this.captcha_id = id;
  var url = "http://" + location.hostname + ":18080";
  this.setImageUrl(url + "/captcha/" + id + ".png");
}

Captcha.prototype.setImageUrl = function(url) {
  this.widget.image.setAttribute("src", url);
}

/**
 * @brief process captcha form
 */
Captcha.prototype.process = function() {
  console.log("process");
  console.log(this);
  if (this.captcha_id) {
    var solution = this.widget.entry.value;
    var xhr = new XMLHttpRequest();
    var self = this;
    // TODO: https detection
    var url = "http://" + location.hostname + this.prefix + "captcha.json";
    xhr.open('post', url, true);
    xhr.onreadystatechange = function() {
      if (xhr.readyState == 4 && xhr.status == 200 ) {
        var jdata = JSON.parse(xhr.responseText);
        if (jdata.solved) {
          // woot we solved it 
          self.hide();
        } else {
          // TODO: inform user of bad solution
          self.load();
        }
      }
    }
    var postdata = "captchaId="+encodeURIComponent(this.captcha_id);
    postdata += "&captchaSolution="+encodeURIComponent(solution);
    xhr.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
    xhr.send(postdata);
  } else {
    // TODO: inform user of no captcha entred
    self.load();
  }
}

/**
 * @brief hide the captcha pane
 */
Captcha.prototype.hide = function () {
  console.log("hide captcha");
  this.widget.widget.style.zIndex = -1;
}

/**
 * build login widget
 * for captcha / mod login 
 */
function buildLogin(domElem) {
  var widget = document.createElement("div");
  widget.className = "livechan_login_widget";
  widget.style.zIndex = -1;
  

  var mod_div = document.createElement("div");
  mod_div.className = "livechan_login";

  var mod_form = document.createElement("form");
  
  var mod_username = document.createElement("input");
  mod_form.appendChild(mod_username);
  mod_username.className = "livechan_login_username";
  
  var mod_password = document.createElement("input");
  mod_password.className = "livechan_login_password";
  mod_password.setAttribute("type", "password");
  mod_form.appendChild(mod_password);
  
  var mod_submit = document.createElement("input");
  mod_password.className = "livechan_login_submit";
  mod_submit.setAttribute("type", "submit");
  mod_submit.setAttribute("value", "login");
  mod_form.appendChild(mod_submit);
  mod_div.appendChild(mod_form);
  widget.appendChild(mod_div);
  domElem.appendChild(widget);
  return {
    widget: widget,
    mod: {
      form: mod_form,
      username: mod_username,
      password: mod_password,
      submit: mod_submit
    }
  }
}


function Login(domElem) {
  this._login = buildLogin(domElem);
}

Login.prototype.show = function() {
  var self = this;
  self._login.widget.style.zIndex = 5;
  console.log("show login widget");
}
