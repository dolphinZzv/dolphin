// Copy button on code blocks
(function() {
  document.querySelectorAll('pre').forEach(function(pre) {
    var btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'Copy';
    btn.addEventListener('click', function() {
      var code = pre.querySelector('code');
      var text = code ? code.textContent : pre.textContent;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(function() {
          btn.textContent = 'Copied!';
          setTimeout(function() { btn.textContent = 'Copy'; }, 2000);
        });
      } else {
        var ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        ta.style.left = '-9999px';
        document.body.appendChild(ta);
        ta.select();
        try {
          document.execCommand('copy');
          btn.textContent = 'Copied!';
          setTimeout(function() { btn.textContent = 'Copy'; }, 2000);
        } catch (e) {
          btn.textContent = 'Error';
        }
        document.body.removeChild(ta);
      }
    });
    pre.appendChild(btn);
  });

  // Activate first tab in each tab group
  document.querySelectorAll('.tabs').forEach(function(tabs) {
    var first = tabs.querySelector('.tab-header');
    if (first && !tabs.querySelector('.tab-header.active')) {
      first.classList.add('active');
    }
  });

})();

// Tab switching (called from onclick)
function switchTab(btn) {
  var tabs = btn.closest('.tabs');
  var id = btn.getAttribute('data-tab');

  tabs.querySelectorAll('.tab-header').forEach(function(h) { h.classList.remove('active'); });
  btn.classList.add('active');

  tabs.querySelectorAll('.tab-panel').forEach(function(p) { p.style.display = 'none'; });
  var panel = tabs.querySelector('.tab-panel[data-tab="' + id + '"]');
  if (panel) panel.style.display = 'block';
}
