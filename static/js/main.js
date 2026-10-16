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
  initEnfrentamientoUnico();
  initNavMenu();

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

  marcarEnfrentamientoUnicoDetallado();
}

// === MODO "CADA PAREJA SE ENFRENTA UNA VEZ" ===
// Si A venció a B, la casilla "B ganó a A" queda bloqueada (en rojo). Si ya venían
// marcadas las dos (datos antiguos), ambas se señalan como conflicto para corregirlas.
function modoUnicoActivo(formId) {
  const cb = document.querySelector('.modo-unico[data-form="' + formId + '"]');
  return !!(cb && cb.checked);
}

// Aplica el estado bloqueado/conflicto a un input según su victoria inversa
function aplicarBloqueoInverso(input, inverso, activo, marcador) {
  const inversoMarcado = !!(inverso && inverso.checked);
  const bloqueado = activo && inversoMarcado && !input.checked;
  const conflicto = activo && inversoMarcado && input.checked;
  if (bloqueado) input.disabled = true;
  marcador.classList.toggle('bloqueado', bloqueado);
  marcador.classList.toggle('conflicto', conflicto);
  marcador.title = bloqueado ? 'Ya perdió contra este jugador: solo se enfrentan una vez'
    : conflicto ? 'Conflicto: los dos aparecen ganándose mutuamente' : marcador.dataset.titulo || '';
}

function marcarEnfrentamientoUnicoDetallado() {
  const form = document.getElementById('form-resultados');
  if (!form) return;
  const activo = modoUnicoActivo('form-resultados');
  form.querySelectorAll('input[type="checkbox"][name^="victoria_"]').forEach(function(input) {
    const label = input.closest('.checkbox-label');
    if (!label) return;
    const jugID = input.name.slice('victoria_'.length);
    const inverso = form.querySelector('input[name="victoria_' + input.value + '"][value="' + jugID + '"]');
    input.disabled = false;
    aplicarBloqueoInverso(input, inverso, activo, label);
  });
}

function initEnfrentamientoUnico() {
  const form = document.getElementById('form-resultados');
  if (!form) return;
  form.addEventListener('change', function(e) {
    if (e.target.name && e.target.name.indexOf('victoria_') === 0) marcarEnfrentamientoUnicoDetallado();
  });
  // Los dos formularios (detallado y matriz) tienen su propio check: se mantienen sincronizados
  const checks = document.querySelectorAll('.modo-unico');
  checks.forEach(function(cb) {
    cb.addEventListener('change', function() {
      checks.forEach(function(otro) { otro.checked = cb.checked; });
      marcarEnfrentamientoUnicoDetallado();
      const matriz = document.getElementById('form-matriz');
      if (matriz) matriz.dispatchEvent(new Event('change'));
    });
  });
}

// === MENÚ DESPLEGABLE DEL NAV ===
// Cierra el <details> al pulsar fuera de él
function initNavMenu() {
  document.addEventListener('click', function(e) {
    document.querySelectorAll('details.nav-menu[open]').forEach(function(menu) {
      if (!menu.contains(e.target)) menu.removeAttribute('open');
    });
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
    const activo = modoUnicoActivo('form-matriz');
    form.querySelectorAll('.matriz-vict').forEach(function(inp) {
      const rowId = inp.closest('tr').dataset.jug;
      const colId = inp.value;
      const ok = juega[rowId] && juega[colId];
      inp.disabled = !ok;
      if (!ok) inp.checked = false;
    });
    // Segunda pasada (ya con todas las casillas actualizadas): bloquear victorias inversas
    form.querySelectorAll('.matriz-vict').forEach(function(inp) {
      const rowId = inp.closest('tr').dataset.jug;
      const inverso = form.querySelector('tr[data-jug="' + inp.value + '"] .matriz-vict[value="' + rowId + '"]');
      const td = inp.closest('td');
      if (!td.dataset.titulo) td.dataset.titulo = '';
      aplicarBloqueoInverso(inp, inverso, activo, td);
    });
    // Colores y notas: habilitados solo si la fila juega
    form.querySelectorAll('.matriz-color-input, .matriz-notas').forEach(function(inp) {
      const rowId = inp.closest('tr').dataset.jug;
      inp.disabled = !juega[rowId];
    });
  }

  // Cualquier cambio (quién juega, victorias o el modo de enfrentamiento) recalcula la matriz
  form.addEventListener('change', refrescar);
  refrescar();

  // Alternar entre vista detallada y matriz
  const btn = document.getElementById('btn-vista-matriz');
  const detallada = document.getElementById('form-resultados');
  if (btn && detallada) {
    btn.addEventListener('click', function() {
      const mostrarMatriz = form.style.display === 'none';
      // Los dos formularios envían a la misma URL: antes de cambiar se copia lo editado
      // en la vista visible a la otra, para no guardar después datos antiguos
      if (mostrarMatriz) copiarDetalladaAMatriz(detallada, form);
      else copiarMatrizADetallada(form, detallada);
      form.style.display = mostrarMatriz ? 'block' : 'none';
      detallada.style.display = mostrarMatriz ? 'none' : 'block';
      btn.textContent = mostrarMatriz ? '📝 Vista detallada' : '⚡ Vista rápida (matriz)';
      if (mostrarMatriz) refrescar();
    });
  }
}

// Copia el estado (marcado y clase visual) de las casillas victoria_*/color_* y las
// notas de un formulario al otro. Las casillas se emparejan por name + value.
function copiarCampos(origen, destino) {
  destino.querySelectorAll('input[type="checkbox"][name^="victoria_"], input[type="checkbox"][name^="color_"]').forEach(function(inp) {
    const par = origen.querySelector('input[name="' + inp.name + '"][value="' + inp.value + '"]');
    inp.checked = !!(par && par.checked);
    const label = inp.closest('.checkbox-label, .mana-checkbox');
    if (label) label.classList.toggle('checked', inp.checked);
  });
  destino.querySelectorAll('input[type="text"][name^="notas_"]').forEach(function(inp) {
    const par = origen.querySelector('input[name="' + inp.name + '"]');
    if (par) inp.value = par.value;
  });
}

function copiarDetalladaAMatriz(detallada, matriz) {
  const ids = document.getElementById('jugadores-sesion-ids').value.split(',').filter(Boolean);
  matriz.querySelectorAll('.matriz-juega').forEach(function(cb) {
    cb.checked = ids.indexOf(cb.dataset.jug) !== -1;
  });
  copiarCampos(detallada, matriz);
  const desc = detallada.querySelector('[name="descripcion"]');
  const descMatriz = matriz.querySelector('[name="descripcion"]');
  if (desc && descMatriz) descMatriz.value = desc.value;
}

function copiarMatrizADetallada(matriz, detallada) {
  const actuales = document.getElementById('jugadores-sesion-ids').value.split(',').filter(Boolean);
  // Igualar quién juega sin que cada cambio desmarque victorias a mitad de la copia
  const original = window.actualizarRivalesVisibles;
  window.actualizarRivalesVisibles = function() {};
  matriz.querySelectorAll('.matriz-juega').forEach(function(cb) {
    const btn = document.getElementById('btn-jug-' + cb.dataset.jug);
    if (btn && cb.checked !== (actuales.indexOf(cb.dataset.jug) !== -1)) {
      toggleJugadorSesion(parseInt(cb.dataset.jug, 10), btn);
    }
  });
  window.actualizarRivalesVisibles = original;
  copiarCampos(matriz, detallada);
  actualizarRivalesVisibles();
}
