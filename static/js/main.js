// === MTG Tracker — Scripts de interactividad ===

// Activar/desactivar estilo visual de checkboxes personalizados
document.addEventListener('DOMContentLoaded', () => {
  // Registrar el service worker (PWA instalable + carga rápida)
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/static/sw.js').catch(() => {});
  }

  // Tablas ordenables (ranking) y modo matriz de resultados
  initSortableTables();
  initMatrizResultados();

  // Checkboxes personalizados (colores maná y rivales)
  document.querySelectorAll('.mana-checkbox, .checkbox-label').forEach(label => {
    const input = label.querySelector('input[type="checkbox"]');
    if (!input) return;

    // Estado inicial
    if (input.checked) label.classList.add('checked');

    label.addEventListener('click', () => {
      // El click en el label cambia el estado del input automáticamente
      setTimeout(() => {
        label.classList.toggle('checked', input.checked);
      }, 0);
    });
  });

  // Evitar que un jugador se marque como rival de sí mismo en victorias
  document.querySelectorAll('[data-self-id]').forEach(form => {
    const selfId = form.dataset.selfId;
    const inputs = form.querySelectorAll(`input[value="${selfId}"]`);
    inputs.forEach(inp => {
      const parent = inp.closest('.checkbox-label');
      if (parent) {
        parent.style.opacity = '0.3';
        parent.style.pointerEvents = 'none';
      }
      inp.disabled = true;
    });
  });

  // Colapsables en el formulario de resultados
  document.querySelectorAll('.resultado-form-header').forEach(header => {
    header.addEventListener('click', () => {
      const body = header.nextElementSibling;
      if (!body) return;
      const isOpen = body.style.display !== 'none';
      body.style.display = isOpen ? 'none' : 'block';
      const icon = header.querySelector('.toggle-icon');
      if (icon) icon.textContent = isOpen ? '▶' : '▼';
    });
  });
});

// Manejar selección de jugadores para la sesión en el formulario de resultados
function toggleJugadorSesion(jugadorId, btn) {
  const jugadoresInput = document.getElementById('jugadores-sesion-ids');
  const resultadoCard = document.getElementById('resultado-card-' + jugadorId);

  const actuales = jugadoresInput.value
    ? jugadoresInput.value.split(',').filter(Boolean)
    : [];

  const idx = actuales.indexOf(String(jugadorId));
  if (idx === -1) {
    // Añadir
    actuales.push(String(jugadorId));
    btn.style.borderColor = 'var(--border-hover)';
    btn.style.color = 'var(--text)';
    btn.style.background = 'var(--bg-surface)';
    btn.style.fontWeight = '500';
    if (resultadoCard) resultadoCard.style.display = 'block';
  } else {
    // Quitar
    actuales.splice(idx, 1);
    btn.style.borderColor = '';
    btn.style.color = '';
    btn.style.background = '';
    btn.style.fontWeight = '';
    if (resultadoCard) resultadoCard.style.display = 'none';
  }

  jugadoresInput.value = actuales.join(',');

  // Actualizar visibilidad de las opciones de "Ganó a..." en TODOS los formularios:
  // un jugador que no juega la sesión no puede aparecer como rival vencido
  actualizarRivalesVisibles();
}

// Oculta/muestra los checkboxes de rivales según los jugadores que juegan la sesión
function actualizarRivalesVisibles() {
  const jugadoresInput = document.getElementById('jugadores-sesion-ids');
  if (!jugadoresInput) return;

  const enSesion = jugadoresInput.value
    ? jugadoresInput.value.split(',').filter(Boolean)
    : [];

  // Recorrer todos los checkbox de tipo "victoria_*" que estén dentro de un
  // .checkbox-label (los de la vista detallada; la matriz se gestiona aparte)
  document.querySelectorAll('input[type="checkbox"][name^="victoria_"]').forEach(function(input) {
    const rivalID = input.value;
    const label = input.closest('.checkbox-label');
    if (!label) return;

    if (enSesion.includes(rivalID)) {
      // El rival juega la sesión: visible
      label.style.display = '';
    } else {
      // El rival no juega: ocultar y desmarcar
      label.style.display = 'none';
      if (input.checked) {
        input.checked = false;
        label.classList.remove('checked');
      }
    }
  });
}

// === TEMA CLARO/OSCURO ===
// Alterna el tema y lo guarda. El script inline del <head> lo aplica antes de pintar.
function toggleTema() {
  const root = document.documentElement;
  let actual = root.dataset.theme;
  if (!actual) {
    actual = (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light';
  }
  const nuevo = actual === 'dark' ? 'light' : 'dark';
  root.dataset.theme = nuevo;
  try { localStorage.setItem('tema', nuevo); } catch (e) {}
}

// === ORDENAR TABLAS POR COLUMNA ===
// Cualquier tabla .sortable-table cuyas <th> tengan la clase .sortable se ordena
// al hacer clic en la cabecera. Usa data-val en las celdas para números/porcentajes.
function initSortableTables() {
  document.querySelectorAll('table.sortable-table').forEach(function(table) {
    table.querySelectorAll('thead th.sortable').forEach(function(th) {
      th.addEventListener('click', function() { sortTableByHeader(table, th); });
    });
  });
}

function sortTableByHeader(table, th) {
  const colIndex = th.cellIndex;
  const type = th.dataset.sort === 'num' ? 'num' : 'text';
  const tbody = table.querySelector('tbody');
  if (!tbody) return;

  // Dirección: si ya está activa, alterna; si no, num descendente / texto ascendente.
  let asc;
  if (th.classList.contains('sort-asc')) asc = false;
  else if (th.classList.contains('sort-desc')) asc = true;
  else asc = (type === 'text');

  const valorDe = function(row) {
    const cell = row.children[colIndex];
    if (!cell) return type === 'num' ? 0 : '';
    const raw = cell.dataset.val !== undefined ? cell.dataset.val : cell.textContent.trim();
    if (type === 'num') { const n = parseFloat(raw); return isNaN(n) ? 0 : n; }
    return raw.toLowerCase();
  };

  const rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
  rows.sort(function(a, b) {
    const va = valorDe(a), vb = valorDe(b);
    if (va < vb) return asc ? -1 : 1;
    if (va > vb) return asc ? 1 : -1;
    return 0;
  });
  rows.forEach(function(r) { tbody.appendChild(r); });

  table.querySelectorAll('thead th').forEach(function(h) { h.classList.remove('sort-asc', 'sort-desc'); });
  th.classList.add(asc ? 'sort-asc' : 'sort-desc');

  // Renumerar la columna de posición (#) según el nuevo orden visible
  tbody.querySelectorAll('tr .rank-num').forEach(function(c, i) { c.textContent = i + 1; });
}

// === MODO MATRIZ DE RESULTADOS ===
// Vista rápida: una casilla (fila, columna) = "el jugador de la fila venció al de la
// columna". Genera el mismo POST que la vista detallada.
function initMatrizResultados() {
  const form = document.getElementById('form-matriz');
  if (!form) return;
  const hidden = document.getElementById('matriz-jugadores');
  const juegaBoxes = form.querySelectorAll('.matriz-juega');

  function refrescar() {
    const juega = {};
    juegaBoxes.forEach(function(cb) { if (cb.checked) juega[cb.dataset.jug] = true; });
    hidden.value = Object.keys(juega).join(',');

    form.querySelectorAll('tr[data-jug]').forEach(function(tr) {
      tr.classList.toggle('fila-inactiva', !juega[tr.dataset.jug]);
    });
    // Una victoria solo es válida si juegan tanto la fila como la columna
    form.querySelectorAll('.matriz-vict').forEach(function(inp) {
      const rowId = inp.closest('tr').dataset.jug;
      const colId = inp.value;
      const ok = juega[rowId] && juega[colId];
      inp.disabled = !ok;
      if (!ok) inp.checked = false;
    });
    // Colores y notas: habilitados solo si la fila juega
    form.querySelectorAll('.matriz-color-input, .matriz-notas').forEach(function(inp) {
      const rowId = inp.closest('tr').dataset.jug;
      inp.disabled = !juega[rowId];
    });
  }

  juegaBoxes.forEach(function(cb) { cb.addEventListener('change', refrescar); });
  refrescar();

  // Alternar entre vista detallada y matriz
  const btn = document.getElementById('btn-vista-matriz');
  const detallada = document.getElementById('form-resultados');
  if (btn && detallada) {
    btn.addEventListener('click', function() {
      const mostrarMatriz = form.style.display === 'none';
      form.style.display = mostrarMatriz ? 'block' : 'none';
      detallada.style.display = mostrarMatriz ? 'none' : 'block';
      btn.textContent = mostrarMatriz ? '📝 Vista detallada' : '⚡ Vista rápida (matriz)';
      if (mostrarMatriz) refrescar();
    });
  }
}
