
function doGet(e) {
  return ContentService.createTextOutput(JSON.stringify({status:'ok'}))
    .setMimeType(ContentService.MimeType.JSON);
}

function doPost(e) {
  var ss = SpreadsheetApp.openById("1H0qsC1G1BPu-4xzILvP4XH1DfOWbEBkIR63pDdDImQM"); // substituir
  var sheet = ss.getSheetByName("Leads") || ss.getSheets()[0];

  // Tenta parsear JSON; se não der, usa e.parameter (form-encoded)
  var data = {};
  try {
    if (e.postData && e.postData.contents) {
      var ct = (e.postData.type || "");
      if (ct.indexOf("application/json") !== -1) {
        data = JSON.parse(e.postData.contents);
      } else {
        data = e.parameter; // x-www-form-urlencoded ou multipart/form-data
      }
    } else {
      data = e.parameter || {};
    }
  } catch(err) {
    data = e.parameter || {};
  }

  // Assegura campos mínimos
  var nome = data.nome || data.name || "";
  var email = data.email || "";
  var telefone = data.telefone || data.tel || "";
  var empresa = data.empresa || "";
  var assunto = data.assunto || "";
  var mensagem = data.mensagem || data.message || "";

  sheet.appendRow([ new Date(), nome, email, telefone, empresa, assunto, mensagem ]);

  var output = ContentService.createTextOutput(JSON.stringify({status: 'success'}));
  output.setMimeType(ContentService.MimeType.JSON);
  return output;
}
