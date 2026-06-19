// State
const state = {
    currentTab: 'dashboard',
    scanId: null,
    eventSource: null,
    theme: 'dark',
    chart: null,
    currentResults: null, // Store current results for report actions
};

// DOM Elements
const elements = {
    tabs: document.querySelectorAll('.nav-btn'),
    tabContents: document.querySelectorAll('.tab-content'),
    dropZone: document.getElementById('dropZone'),
    fileInput: document.getElementById('fileInput'),
    gitUrl: document.getElementById('gitUrl'),
    scanGitBtn: document.getElementById('scanGitBtn'),
    uploadProgress: document.getElementById('uploadProgress'),
    uploadProgressBar: document.getElementById('uploadProgressBar'),
    uploadStatus: document.getElementById('uploadStatus'),
    scanProgress: document.getElementById('scanProgress'),
    scanProgressBar: document.getElementById('scanProgressBar'),
    scanStatus: document.getElementById('scanStatus'),
    scanLog: document.getElementById('scanLog'),
    scanResults: document.getElementById('scanResults'),
    findingsList: document.getElementById('findingsList'),
    resultsSummary: document.getElementById('resultsSummary'),
    themeToggle: document.getElementById('themeToggle'),
};

// Tab Navigation
elements.tabs.forEach(tab => {
    tab.addEventListener('click', () => {
        const tabName = tab.dataset.tab;
        switchTab(tabName);
    });
});

function switchTab(tabName) {
    state.currentTab = tabName;
    
    elements.tabs.forEach(t => t.classList.remove('active'));
    elements.tabContents.forEach(c => c.classList.remove('active'));
    
    document.querySelector(`.nav-btn[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(tabName).classList.add('active');
    
    if (tabName === 'dashboard') {
        loadDashboard();
    } else if (tabName === 'history') {
        loadHistory();
    }
}

// ==================== DASHBOARD ====================

async function loadDashboard() {
    try {
        const statsRes = await fetch('/api/dashboard/stats');
        if (!statsRes.ok) {
            throw new Error(`HTTP ${statsRes.status}`);
        }
        const stats = await statsRes.json();
        
        // Update stats with safe fallbacks
        document.getElementById('totalScans').textContent = stats.total_scans || 0;
        document.getElementById('totalFindings').textContent = stats.total_findings || 0;
        document.getElementById('criticalCount').textContent = stats.critical || 0;
        document.getElementById('highCount').textContent = stats.high || 0;
        document.getElementById('mediumCount').textContent = stats.medium || 0;
        document.getElementById('lowCount').textContent = stats.low || 0;
        
        // Load recent findings
        const historyRes = await fetch('/api/dashboard/history');
        if (historyRes.ok) {
            const history = await historyRes.json();
            renderRecentFindings(history);
            updateChart(history);
        }
    } catch (error) {
        console.error('Failed to load dashboard:', error);
        // Show fallback values
        document.getElementById('totalScans').textContent = '0';
        document.getElementById('totalFindings').textContent = '0';
        document.getElementById('criticalCount').textContent = '0';
        document.getElementById('highCount').textContent = '0';
        document.getElementById('mediumCount').textContent = '0';
        document.getElementById('lowCount').textContent = '0';
    }
}

function renderRecentFindings(history) {
    const container = document.getElementById('recentFindings');
    
    // Extract findings from history
    const findings = [];
    if (Array.isArray(history)) {
        history.forEach(item => {
            if (item.findings && Array.isArray(item.findings)) {
                findings.push(...item.findings);
            }
        });
    }
    
    if (findings.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-check-circle" style="font-size: 2rem; color: #2ed573;"></i>
                <p>No findings yet. Run your first scan!</p>
            </div>
        `;
        return;
    }
    
    // Get critical findings
    const criticalFindings = findings.filter(f => f.severity === 'critical').slice(0, 5);
    
    if (criticalFindings.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-shield-alt" style="font-size: 2rem; color: #2ed573;"></i>
                <p>No critical findings! Your code looks secure! 🎉</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = criticalFindings.map(f => `
        <div class="finding-item">
            <span class="finding-severity severity-${f.severity || 'low'}">${f.severity || 'info'}</span>
            <div class="finding-content">
                <div class="finding-title">${f.title || 'Unknown Issue'}</div>
                <div class="finding-file">📁 ${f.file || 'unknown'}:${f.line || 0}</div>
                ${f.description ? `<div class="finding-description">${f.description}</div>` : ''}
                ${f.suggestion ? `<div class="finding-suggestion">💡 ${f.suggestion}</div>` : ''}
            </div>
        </div>
    `).join('');
}

function updateChart(history) {
    const ctx = document.getElementById('historyChart').getContext('2d');
    
    // Destroy existing chart if it exists
    if (state.chart) {
        state.chart.destroy();
    }
    
    // Prepare data
    const dates = [];
    const counts = [];
    
    if (Array.isArray(history) && history.length > 0) {
        // Take last 30 entries or all if less
        const recentHistory = history.slice(-30);
        recentHistory.forEach(item => {
            if (item.date) {
                const date = new Date(item.date);
                dates.push(date.toLocaleDateString());
                counts.push(item.findings || 0);
            }
        });
    }
    
    // If no data, show empty chart
    if (dates.length === 0) {
        dates.push('No Data');
        counts.push(0);
    }
    
    // Get theme colors from computed styles
    const isDark = window.getComputedStyle(document.documentElement)
        .getPropertyValue('--bg-primary').trim() === '#0a0e17' || 
        document.documentElement.style.getPropertyValue('--bg-primary') === '#0a0e17';
    
    // Chart colors based on theme
    const textColor = isDark ? '#8899bb' : '#4a4a6a';
    const gridColor = isDark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)';
    const borderColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)';
    const tooltipBg = isDark ? 'rgba(26, 35, 51, 0.95)' : 'rgba(255, 255, 255, 0.95)';
    const tooltipTitle = isDark ? '#e8edf5' : '#1a1a2e';
    const tooltipBody = isDark ? '#8899bb' : '#4a4a6a';
    const tooltipBorder = isDark ? '#1e2d42' : '#e0e4ea';
    
    state.chart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: dates,
            datasets: [{
                label: 'Findings',
                data: counts,
                borderColor: '#4f8cff',
                backgroundColor: 'rgba(79, 140, 255, 0.1)',
                fill: true,
                tension: 0.4,
                pointBackgroundColor: '#4f8cff',
                pointBorderColor: isDark ? '#1a2333' : '#ffffff',
                pointBorderWidth: 2,
                pointRadius: 4,
                pointHoverRadius: 6,
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: false,
                },
                tooltip: {
                    backgroundColor: tooltipBg,
                    titleColor: tooltipTitle,
                    bodyColor: tooltipBody,
                    borderColor: tooltipBorder,
                    borderWidth: 1,
                    cornerRadius: 8,
                    padding: 12,
                    titleFont: {
                        size: 13,
                        weight: '600',
                    },
                    bodyFont: {
                        size: 12,
                    },
                    callbacks: {
                        label: function(context) {
                            return `Findings: ${context.parsed.y}`;
                        },
                        title: function(context) {
                            return context[0].label;
                        }
                    }
                }
            },
            scales: {
                x: {
                    grid: {
                        color: gridColor,
                        drawBorder: false,
                    },
                    ticks: {
                        color: textColor,
                        maxTicksLimit: 10,
                        font: {
                            size: 11,
                            weight: '400',
                        },
                    }
                },
                y: {
                    grid: {
                        color: gridColor,
                        drawBorder: false,
                    },
                    ticks: {
                        color: textColor,
                        stepSize: 1,
                        font: {
                            size: 11,
                            weight: '400',
                        },
                    },
                    beginAtZero: true,
                }
            },
            elements: {
                line: {
                    borderWidth: 2,
                },
                point: {
                    radius: 3,
                    hoverRadius: 5,
                }
            },
            interaction: {
                intersect: false,
                mode: 'index',
            },
            animation: {
                duration: 800,
                easing: 'easeOutQuart'
            }
        }
    });
}

// ==================== SCAN FUNCTIONS ====================

// File Upload
elements.dropZone.addEventListener('click', () => {
    elements.fileInput.click();
});

elements.dropZone.addEventListener('dragover', (e) => {
    e.preventDefault();
    elements.dropZone.classList.add('dragover');
});

elements.dropZone.addEventListener('dragleave', () => {
    elements.dropZone.classList.remove('dragover');
});

elements.dropZone.addEventListener('drop', (e) => {
    e.preventDefault();
    elements.dropZone.classList.remove('dragover');
    
    if (e.dataTransfer.files.length > 0) {
        handleFileUpload(e.dataTransfer.files);
    }
});

elements.fileInput.addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
        handleFileUpload(e.target.files);
    }
});

async function handleFileUpload(files) {
    const formData = new FormData();
    
    // If it's a single file or folder, add all files
    for (let i = 0; i < files.length; i++) {
        formData.append('files[]', files[i]);
    }
    
    elements.uploadProgress.style.display = 'block';
    elements.uploadProgressBar.style.width = '30%';
    elements.uploadStatus.textContent = `Uploading ${files.length} files...`;
    
    try {
        const response = await fetch('/api/scan/upload', {
            method: 'POST',
            body: formData,
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Upload failed');
        }
        
        const result = await response.json();
        
        if (result.scan_id) {
            elements.uploadProgressBar.style.width = '100%';
            elements.uploadStatus.textContent = 'Upload complete! Starting scan...';
            state.scanId = result.scan_id;
            startScanProgress(result.scan_id);
        }
    } catch (error) {
        console.error('Upload failed:', error);
        elements.uploadStatus.textContent = `❌ Upload failed: ${error.message}`;
        elements.uploadProgressBar.style.width = '0%';
    }
}

// Git Scan
elements.scanGitBtn.addEventListener('click', async () => {
    const url = elements.gitUrl.value.trim();
    if (!url) {
        alert('Please enter a Git repository URL');
        return;
    }
    
    elements.scanGitBtn.disabled = true;
    elements.scanGitBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Cloning...';
    
    try {
        const response = await fetch('/api/scan/git', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url }),
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.message || 'Git scan failed');
        }
        
        const result = await response.json();
        
        if (result.scan_id) {
            state.scanId = result.scan_id;
            startScanProgress(result.scan_id);
        }
    } catch (error) {
        console.error('Git scan failed:', error);
        alert(`Failed to scan repository: ${error.message}`);
    } finally {
        elements.scanGitBtn.disabled = false;
        elements.scanGitBtn.innerHTML = '<i class="fas fa-code-branch"></i> Scan Repository';
    }
});

// ==================== SCAN PROGRESS ====================

function startScanProgress(scanId) {
    elements.scanProgress.style.display = 'block';
    elements.scanResults.style.display = 'none';
    elements.scanLog.innerHTML = '';
    
    if (state.eventSource) {
        state.eventSource.close();
    }
    
    state.eventSource = new EventSource(`/api/scan/${scanId}/stream`);
    
    state.eventSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            updateScanProgress(data);
        } catch (e) {
            console.error('Failed to parse progress:', e);
        }
    };
    
    state.eventSource.onerror = () => {
        state.eventSource.close();
        // Try to fetch results
        fetchScanResults(scanId);
    };
}

function updateScanProgress(data) {
    elements.scanProgressBar.style.width = `${data.progress}%`;
    elements.scanStatus.textContent = data.message;
    
    // Add to log with timestamp
    const logEntry = document.createElement('div');
    logEntry.className = 'log-entry';
    const time = new Date().toLocaleTimeString();
    logEntry.innerHTML = `<span class="log-time">[${time}]</span> ${data.message}`;
    elements.scanLog.appendChild(logEntry);
    elements.scanLog.scrollTop = elements.scanLog.scrollHeight;
    
    if (data.status === 'completed' || data.status === 'failed') {
        state.eventSource.close();
        fetchScanResults(state.scanId);
    }
}

async function fetchScanResults(scanId) {
    try {
        const response = await fetch(`/api/scan/${scanId}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const results = await response.json();
        displayScanResults(results);
    } catch (error) {
        console.error('Failed to fetch results:', error);
        elements.scanStatus.textContent = '❌ Failed to load results';
    }
}

function displayScanResults(results) {
    // Store results for report actions
    state.currentResults = results;
    
    elements.scanResults.style.display = 'block';
    elements.scanProgress.style.display = 'none';
    
    // Summary
    const summary = results.summary || { critical: 0, high: 0, medium: 0, low: 0, total: 0 };
    elements.resultsSummary.innerHTML = `
        <span class="summary-item"><span class="count" style="color:var(--critical)">${summary.critical}</span> Critical</span>
        <span class="summary-item"><span class="count" style="color:var(--high)">${summary.high}</span> High</span>
        <span class="summary-item"><span class="count" style="color:var(--medium)">${summary.medium}</span> Medium</span>
        <span class="summary-item"><span class="count" style="color:var(--low)">${summary.low}</span> Low</span>
        <span class="summary-item"><strong>Total: ${summary.total}</strong></span>
    `;
    
    // Findings
    const findings = results.findings || [];
    if (findings.length === 0) {
        elements.findingsList.innerHTML = `
            <div class="empty-state">
                <i class="fas fa-check-circle" style="font-size: 3rem; color: #2ed573;"></i>
                <p style="font-size: 1.2rem; margin-top: 0.5rem;">🎉 No vulnerabilities found!</p>
                <p style="color: var(--text-muted);">Your code looks secure.</p>
            </div>
        `;
        return;
    }
    
    // Sort by severity
    const severityOrder = { critical: 0, high: 1, medium: 2, low: 3 };
    findings.sort((a, b) => (severityOrder[a.severity] || 4) - (severityOrder[b.severity] || 4));
    
    elements.findingsList.innerHTML = findings.map(f => `
        <div class="finding-item">
            <span class="finding-severity severity-${f.severity || 'low'}">${f.severity || 'info'}</span>
            <div class="finding-content">
                <div class="finding-title">${f.title || 'Unknown Issue'}</div>
                <div class="finding-file">📁 ${f.file || 'unknown'}${f.line > 0 ? `:${f.line}` : ''}</div>
                ${f.description ? `<div class="finding-description">${f.description}</div>` : ''}
                ${f.suggestion ? `<div class="finding-suggestion">💡 ${f.suggestion}</div>` : ''}
            </div>
            <div class="finding-actions">
                <button class="btn-small primary" onclick="ignoreFinding('${f.id}')">
                    <i class="fas fa-check"></i> Ignore
                </button>
            </div>
        </div>
    `).join('');
}

// ==================== REPORT FUNCTIONS ====================

// View Report - opens JSON report in new tab
window.viewReport = function(scanId) {
    if (!scanId) {
        alert('No scan ID available. Please run a scan first.');
        return;
    }
    
    // Open report in new tab
    window.open(`/api/report/${scanId}`, '_blank');
};

// Download PDF Report
window.downloadPDF = async function(scanId) {
    if (!scanId) {
        alert('No scan ID available. Please run a scan first.');
        return;
    }
    
    try {
        // Show download status
        const btn = document.querySelector('.btn-success');
        if (btn) {
            btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Generating PDF...';
            btn.disabled = true;
        }
        
        // Create a download link
        const link = document.createElement('a');
        link.href = `/api/report/${scanId}/pdf`;
        link.download = `security-report-${scanId.substring(0, 8)}.pdf`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        // Reset button
        if (btn) {
            btn.innerHTML = '<i class="fas fa-file-pdf"></i> Download PDF Report';
            btn.disabled = false;
        }
    } catch (error) {
        console.error('Failed to download PDF:', error);
        alert('Failed to download PDF. Please try again.');
        
        // Reset button
        const btn = document.querySelector('.btn-success');
        if (btn) {
            btn.innerHTML = '<i class="fas fa-file-pdf"></i> Download PDF Report';
            btn.disabled = false;
        }
    }
};

// ==================== HISTORY ====================

async function loadHistory() {
    try {
        const response = await fetch('/api/dashboard/history');
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const history = await response.json();
        
        const container = document.getElementById('historyList');
        if (!history || history.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-inbox" style="font-size: 2rem; color: #556688;"></i>
                    <p>No scans performed yet</p>
                </div>
            `;
            return;
        }
        
        container.innerHTML = history.map(h => `
            <div class="history-item">
                <span class="date">${new Date(h.date).toLocaleString()}</span>
                <span class="findings-count">${h.findings || 0} findings${h.critical ? ` (${h.critical} critical)` : ''}</span>
                <div class="history-actions">
                    <button class="btn-small primary" onclick="viewReport('${h.id}')" title="View Report">
                        <i class="fas fa-eye"></i>
                    </button>
                    <button class="btn-small success" onclick="downloadPDF('${h.id}')" title="Download PDF">
                        <i class="fas fa-file-pdf"></i>
                    </button>
                </div>
                <span class="status ${h.status || 'completed'}">${h.status || 'completed'}</span>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load history:', error);
        document.getElementById('historyList').innerHTML = `
            <div class="empty-state">
                <i class="fas fa-exclamation-triangle" style="font-size: 2rem; color: #ff6b35;"></i>
                <p>Failed to load history</p>
            </div>
        `;
    }
}

// ==================== FINDING ACTIONS ====================

// Ignore Finding
window.ignoreFinding = async function(findingId) {
    if (!confirm('Mark this finding as false positive?')) return;
    
    try {
        const response = await fetch(`/api/findings/${findingId}/ignore`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ reason: 'False positive' }),
        });
        
        if (response.ok) {
            // Reload current scan results
            if (state.scanId) {
                fetchScanResults(state.scanId);
            }
            // Also reload dashboard
            loadDashboard();
        }
    } catch (error) {
        console.error('Failed to ignore finding:', error);
        alert('Failed to ignore finding. Please try again.');
    }
};

// ==================== THEME TOGGLE ====================

elements.themeToggle.addEventListener('click', () => {
    const isDark = document.documentElement.style.getPropertyValue('--bg-primary') === '#0a0e17' || 
                   !document.documentElement.style.getPropertyValue('--bg-primary');
    
    if (isDark) {
        // Light theme
        document.documentElement.style.setProperty('--bg-primary', '#f0f2f5');
        document.documentElement.style.setProperty('--bg-secondary', '#ffffff');
        document.documentElement.style.setProperty('--bg-card', '#ffffff');
        document.documentElement.style.setProperty('--bg-hover', '#f0f2f5');
        document.documentElement.style.setProperty('--bg-footer', '#f0f2f5');
        document.documentElement.style.setProperty('--text-primary', '#1a1a2e');
        document.documentElement.style.setProperty('--text-secondary', '#4a4a6a');
        document.documentElement.style.setProperty('--text-muted', '#8899bb');
        document.documentElement.style.setProperty('--border-color', '#e0e4ea');
        document.documentElement.style.setProperty('--border-light', '#d0d4da');
        elements.themeToggle.innerHTML = '<i class="fas fa-sun"></i>';
    } else {
        // Dark theme
        document.documentElement.style.setProperty('--bg-primary', '#0a0e17');
        document.documentElement.style.setProperty('--bg-secondary', '#111927');
        document.documentElement.style.setProperty('--bg-card', '#1a2333');
        document.documentElement.style.setProperty('--bg-hover', '#243044');
        document.documentElement.style.setProperty('--bg-footer', '#0d1420');
        document.documentElement.style.setProperty('--text-primary', '#e8edf5');
        document.documentElement.style.setProperty('--text-secondary', '#8899bb');
        document.documentElement.style.setProperty('--text-muted', '#556688');
        document.documentElement.style.setProperty('--border-color', '#1e2d42');
        document.documentElement.style.setProperty('--border-light', '#2a3d5a');
        elements.themeToggle.innerHTML = '<i class="fas fa-moon"></i>';
    }
    
    // Update chart if it exists - force re-render with new colors
    if (state.chart) {
        state.chart.destroy();
        state.chart = null;
        // Reload history data to re-render chart
        loadDashboard();
    }
});

// ==================== INITIALIZE ====================

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
});