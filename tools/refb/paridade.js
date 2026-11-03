// Executado via javascript_tool no Chrome, SOMENTE LEITURA: devolve o estilo computado
// dos mesmos seletores no painel B e no v2, para comparação linha a linha.
(function () {
  var PROPS = ['font-family', 'font-size', 'font-weight', 'line-height', 'color', 'background-color',
    'border-top-color', 'border-radius', 'padding-top', 'padding-right', 'padding-bottom', 'padding-left',
    'margin-top', 'margin-bottom', 'width', 'height', 'letter-spacing', 'box-shadow', 'display', 'text-transform'];
  var SELETORES = {
    'body': 'body',
    '.sidebar': '.sidebar',
    '.brand': '.sidebar .brand',
    '.brand .seal-logo': '.brand .seal-logo',
    'nav.tabs item': 'nav.tabs button:not(.active), nav.tabs a:not(.active)',
    'nav.tabs item.active': 'nav.tabs button.active, nav.tabs a.active',
    'nav.tabs .count': 'nav.tabs .count',
    '.main': '.main',
    '.topbar': '.topbar',
    '.title-row': '.title-row',
    '.topbar h1': '.topbar h1',
    '.topbar .sub': '.topbar .sub',
    '.agenda-date-chip': '.agenda-date-chip',
    '.btn-primary': '.btn-primary',
    '.stats': '.stats',
    '.acc-head': '.acc-head:not(.open)',
    '.acc-head.open': '.acc-head.open',
    '.acc-head .count': '.acc-head .count',
    '#content': '#content',
    'table': 'table',
    'th': 'th',
    'td': 'td',
    '.badge': '.badge',
    '.empty': '.empty',
    '.search': '.search',
    '.sync-status': '.sync-status',
    '.sync-status strong': '.sync-status strong',
    '.idx-ticker': '.idx-ticker',
    '.idx-ticker .idxt-status': '.idx-ticker .idxt-status',
    '#drawer': '#drawer',
    '.drawer h2': '.drawer h2',
    '.drawer .field label': '.drawer .field label',
    '.drawer .field input': '.drawer .field input',
    '.drawer-actions': '.drawer-actions',
    '.toast': '#toast'
  };
  var saida = {};
  Object.keys(SELETORES).forEach(function (rotulo) {
    var el = document.querySelector(SELETORES[rotulo]);
    if (!el) { saida[rotulo] = null; return; }
    var cs = getComputedStyle(el), o = {};
    PROPS.forEach(function (p) { o[p] = cs.getPropertyValue(p); });
    saida[rotulo] = o;
  });
  return JSON.stringify(saida);
})()
