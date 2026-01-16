// API base URL
const API_BASE = '/api';

// Load destinations on page load
document.addEventListener('DOMContentLoaded', () => {
    loadDestinations();
});

// Load all destinations
async function loadDestinations() {
    try {
        const response = await fetch(`${API_BASE}/destinations`);
        if (!response.ok) throw new Error('Failed to load destinations');

        const destinations = await response.json();
        renderDestinations(destinations);
    } catch (error) {
        showAlert('Failed to load destinations: ' + error.message, 'error');
    }
}

// Render destinations table
function renderDestinations(destinations) {
    const tbody = document.getElementById('destinations-tbody');

    if (destinations.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="loading">No destinations configured</td></tr>';
        return;
    }

    tbody.innerHTML = destinations.map(dest => `
        <tr>
            <td><strong>${escapeHtml(dest.name)}</strong></td>
            <td class="url-cell" title="${escapeHtml(dest.url)}">${escapeHtml(dest.url)}</td>
            <td>${escapeHtml(dest.method)}</td>
            <td class="filter-count">${dest.filters ? dest.filters.length : 0} filters</td>
            <td>
                <span class="enabled-badge ${dest.enabled ? 'yes' : 'no'}">
                    ${dest.enabled ? 'Enabled' : 'Disabled'}
                </span>
            </td>
            <td>
                <button class="btn btn-sm btn-primary" onclick="toggleDestination('${dest.id}', ${!dest.enabled})">
                    ${dest.enabled ? 'Disable' : 'Enable'}
                </button>
                <button class="btn btn-sm btn-secondary" onclick="editDestination('${dest.id}')">Edit</button>
                <button class="btn btn-sm btn-danger" onclick="deleteDestination('${dest.id}', '${escapeHtml(dest.name)}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

// Show add modal
function showAddModal() {
    document.getElementById('modal-title').textContent = 'Add Destination';
    document.getElementById('destination-form').reset();
    document.getElementById('dest-id').value = '';
    document.getElementById('dest-headers').value = '{}';
    document.getElementById('dest-filters').value = '[]';
    document.getElementById('modal').style.display = 'block';
}

// Show edit modal
async function editDestination(id) {
    try {
        const response = await fetch(`${API_BASE}/destinations/${id}`);
        if (!response.ok) throw new Error('Failed to load destination');

        const dest = await response.json();

        document.getElementById('modal-title').textContent = 'Edit Destination';
        document.getElementById('dest-id').value = dest.id;
        document.getElementById('dest-name').value = dest.name;
        document.getElementById('dest-url').value = dest.url;
        document.getElementById('dest-method').value = dest.method;
        document.getElementById('dest-timeout').value = dest.timeout_seconds || 30;
        document.getElementById('dest-headers').value = JSON.stringify(dest.headers || {}, null, 2);
        document.getElementById('dest-filters').value = JSON.stringify(dest.filters || [], null, 2);
        document.getElementById('dest-enabled').checked = dest.enabled;

        document.getElementById('modal').style.display = 'block';
    } catch (error) {
        showAlert('Failed to load destination: ' + error.message, 'error');
    }
}

// Close modal
function closeModal() {
    document.getElementById('modal').style.display = 'none';
}

// Handle form submission
document.getElementById('destination-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const id = document.getElementById('dest-id').value;
    const isEdit = id !== '';

    // Parse JSON fields
    let headers, filters;
    try {
        headers = JSON.parse(document.getElementById('dest-headers').value);
    } catch (error) {
        showAlert('Invalid JSON in headers field', 'error');
        return;
    }

    try {
        filters = JSON.parse(document.getElementById('dest-filters').value);
    } catch (error) {
        showAlert('Invalid JSON in filters field', 'error');
        return;
    }

    const destination = {
        id: id || undefined,
        name: document.getElementById('dest-name').value,
        url: document.getElementById('dest-url').value,
        method: document.getElementById('dest-method').value,
        timeout_seconds: parseInt(document.getElementById('dest-timeout').value),
        headers: headers,
        filters: filters,
        enabled: document.getElementById('dest-enabled').checked,
        retry_policy: {
            max_attempts: 10,
            backoff_schedule: [60, 300, 900, 3600, 21600, 86400]
        }
    };

    try {
        const url = isEdit ? `${API_BASE}/destinations/${id}` : `${API_BASE}/destinations`;
        const method = isEdit ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(destination)
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to save destination');
        }

        showAlert(`Destination ${isEdit ? 'updated' : 'created'} successfully`, 'success');
        closeModal();
        loadDestinations();
    } catch (error) {
        showAlert('Failed to save destination: ' + error.message, 'error');
    }
});

// Toggle destination enabled/disabled
async function toggleDestination(id, enabled) {
    try {
        const response = await fetch(`${API_BASE}/destinations/${id}/toggle`, {
            method: 'PATCH'
        });

        if (!response.ok) throw new Error('Failed to toggle destination');

        showAlert(`Destination ${enabled ? 'enabled' : 'disabled'} successfully`, 'success');
        loadDestinations();
    } catch (error) {
        showAlert('Failed to toggle destination: ' + error.message, 'error');
    }
}

// Delete destination
async function deleteDestination(id, name) {
    if (!confirm(`Are you sure you want to delete "${name}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/destinations/${id}`, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Failed to delete destination');

        showAlert('Destination deleted successfully', 'success');
        loadDestinations();
    } catch (error) {
        showAlert('Failed to delete destination: ' + error.message, 'error');
    }
}

// Show alert message
function showAlert(message, type) {
    const alert = document.getElementById('alert');
    alert.textContent = message;
    alert.className = `alert alert-${type}`;
    alert.style.display = 'block';

    setTimeout(() => {
        alert.style.display = 'none';
    }, 5000);
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Close modal when clicking outside
window.onclick = function(event) {
    const modal = document.getElementById('modal');
    if (event.target === modal) {
        closeModal();
    }
}
