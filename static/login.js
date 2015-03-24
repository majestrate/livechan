
/**
 * build login widget
 * for captcha / mod login 
 */
function buildLogin(domElem, chatElem) {
  var widget = document.createElement("div");
  widget.className = "livechan_login_widget";
  
  var captcha_widget = document.createElement("div");
  captcha_widget.className = "livechan_captcha";

  widget.appendChild(captcha_widget);

  var captcha_entry = document.createElement("input");
  var captcha_image = document.createElement("img");

  var mod_div = document.createElement("div");

  var mod_form = document.createElement("form");
  
  var mod_username = document.createElement("input");
  mod_form.appendChild(mod_username);
  mod_password.className = "livechan_login_username";
  
  var mod_password = document.createElement("input");
  mod_password.className = "livechan_login_password";
  mod_password.setAttribute("type", "password");
  mod_form.appendChild(mod_password);
  
  var mod_submit = document.createElement("input");
  mod_password.className = "livechan_login_submit";
  mod_submit.setAttribute("type", "submit");
  mod_submit.setAttribute("value", "login");
  mod_form.appendChild(mod_submit);
  
  
  return {
    captcha: {
      input: captcha_entry,
      image: captcha_image
    },
    mod: {
      form: mod_form,
      username: mod_username,
      password: mod_password,
      submit: mod_submit
    }
  }
}
