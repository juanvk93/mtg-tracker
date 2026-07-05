// === MTG Tracker — Scripts de interactividad ===

// Activar/desactivar estilo visual de checkboxes personalizados
document.addEventListener('DOMContentLoaded', () => {
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

  // Recorrer todos los checkbox de tipo "victoria_*"
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
