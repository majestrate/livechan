//
// welcome page
//


addBoardToList = function(boardName) {
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
      var boardList = JSON.parse(ajax.reponseText);
      for ( var idx = 0 ; idx < boardList; idx ++ ) {
        addBoardToList(boardList[idx]);
      }
    }
  }
  ajax.open("GET", "/channels");
  ajax.send();
});
