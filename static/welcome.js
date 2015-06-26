//
// welcome page
//


addBoardToList = function(boardName) {
  console.log("add board to list: "+boardName);
  var boardList = document.getElementById("boards");
  var elem = document.createElement("div");
  elem.className = "livechan_board_list_entry";
  var link = document.createElement("a");
  link.setAttribute("href", boardName);
  link.appendChild(document.createTextNode(boardName));
  elem.appendChild(link);
  boardList.appendChild(elem);
}

window.addEventListener('load', function() {

  var ajax = new XMLHttpRequest();
  ajax.onreadystatechange = function() {
    if ( ajax.status == 200 && ajax.readyState == XMLHttpRequest.DONE ) {
      console.log(ajax.responseText);
      var boardList = JSON.parse(ajax.responseText);
      for ( var idx = 0 ; idx < boardList; idx ++ ) {
        addBoardToList(boardList[idx]);
      }
    } else {
      console.log(ajax.status);
    }
  }
  ajax.open("GET", "/channels");
  ajax.send();
});
