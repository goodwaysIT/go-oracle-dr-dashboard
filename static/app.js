// --- Global Cache ---
const domCache = {};

// Helper function to get language from URL, defaulting to 'zh'
const getLang = () => new URLSearchParams(window.location.search).get('lang') || 'zh';

// Helper function for translation with fallback
function t(key) {
    return (window.I18N && window.I18N[key]) || key; // Fallback to the key itself if not found
}

// Function to load translations from the API
async function loadTranslations() {
    const lang = getLang();
    if (!lang) {
        // Auto-detect browser language
        const browserLang = (navigator.languages && navigator.languages[0]) || navigator.language || 'zh';
        if (browserLang.startsWith('zh')) {
            lang = 'zh';
        } else if (browserLang.startsWith('ja')) {
            lang = 'ja';
        } else {
            lang = 'en';
        }
    }
    try {
        const response = await fetch(getApiUrl(`api/i18n/${lang}`));
        if (!response.ok) {
            throw new Error(`Failed to fetch translations for ${lang}`);
        }
        window.I18N = await response.json();
    } catch (error) {
        console.error(error);
        window.I18N = {}; // Fallback to an empty object
    }
}

// Function to update titles on the page
function updateTitles(titles) {
    const newTitles = titles || {};
    if (newTitles.main_title) {
        document.title = newTitles.main_title;
        if (domCache.mainTitleH1) {
            domCache.mainTitleH1.textContent = newTitles.main_title;
        }
    }
    if (domCache.prodDcTitle && newTitles.prod_data_center) {
        domCache.prodDcTitle.textContent = newTitles.prod_data_center;
    }
    if (domCache.drDcTitle && newTitles.dr_data_center) {
        domCache.drDcTitle.textContent = newTitles.dr_data_center;
    }
}

// Function to apply layout from config
function applyLayout() {
    const layoutConfig = window.APP_CONFIG && window.APP_CONFIG.layout;
    const columns = layoutConfig && layoutConfig.columns > 0 ? layoutConfig.columns : 2; // Default to 2 columns

    if (domCache.productionContainer) {
        domCache.productionContainer.style.setProperty('--db-grid-columns', columns);
    }
    if (domCache.disasterContainer) {
        domCache.disasterContainer.style.setProperty('--db-grid-columns', columns);
    }
}

// Function to apply translations
function applyTranslations() {
    const translations = window.I18N || {};

    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        if (translations[key]) {
            element.textContent = translations[key];
        }
    });

    if (domCache.lbIpValue && translations.lbIpLoading) {
        domCache.lbIpValue.textContent = translations.lbIpLoading;
    }
}

// Access the base path provided by the Go backend
const API_BASE_PATH = (window.APP_CONFIG && window.APP_CONFIG.basePath) || '/';

// Function to construct full API URLs
function getApiUrl(endpoint) {
    let cleanEndpoint = endpoint;
    if (API_BASE_PATH !== '/' && endpoint.startsWith('/')) {
        cleanEndpoint = endpoint.substring(1);
    }
    let effectiveBasePath = API_BASE_PATH;
    if (effectiveBasePath !== '/' && !effectiveBasePath.endsWith('/')) {
       effectiveBasePath += '/';
    }
    if (effectiveBasePath === '/' && cleanEndpoint.startsWith('/')) {
         return cleanEndpoint;
    }
    return effectiveBasePath + cleanEndpoint;
}

// Format time
function formatTime(timestamp) {
    const date = new Date(timestamp * 1000);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

// Update real-time clock
function updateCurrentTime() {
    const now = new Date();
        const lang = getLang();

    let locale;
    switch (lang) {
        case 'zh':
            locale = 'zh-CN';
            break;
        case 'ja':
            locale = 'ja-JP';
            break;
        default:
            locale = 'en-US';
            break;
    }

    const options = {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
        // Force the Japanese calendar to use the Gregorian system for consistency
        calendar: 'gregory'
    };

    try {
        const formatter = new Intl.DateTimeFormat(locale, options);
        const formattedTime = formatter.format(now);

        if (domCache.currentTime) {
            domCache.currentTime.textContent = formattedTime;
        }
    } catch (error) {
        console.error('Failed to format date, falling back to basic format.', error);
        // Fallback for older browsers
        const year = now.getFullYear();
        const month = (now.getMonth() + 1).toString().padStart(2, '0');
        const day = now.getDate().toString().padStart(2, '0');
        const hours = now.getHours().toString().padStart(2, '0');
        const minutes = now.getMinutes().toString().padStart(2, '0');
        const seconds = now.getSeconds().toString().padStart(2, '0');
        domCache.currentTime.textContent = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
    }
}

// Fetch data and render
async function fetchAndRenderData() {
    try {
                const urlParams = new URLSearchParams(window.location.search);
        const useMockData = urlParams.get('mock') === 'true';
        const lang = getLang();
        const dataUrl = useMockData ? `api/mock-data?lang=${lang}` : 'api/data';

        const response = await fetch(getApiUrl(dataUrl));
        const result = await response.json();

        if (result.code === 200) {
            // If mock data is used and titles are provided, update the UI with them.
            if (useMockData && result.titles) {
                updateTitles(result.titles);
            }
            render(result.data);
        } else {
            showError(result.message || 'Failed to fetch data');
        }
    } catch (error) {
        console.error('Failed to fetch data:', error);
        showError('Failed to fetch or parse data.');
    }
}

function render(data) {
    // --- Dynamic Layout ---
    const dbCount = data.length;
    if (dbCount > 4) { // When there are more than 4 databases, apply wide-screen layout
        domCache.dashboardContainer.classList.add('wide-layout');
    } else {
        domCache.dashboardContainer.classList.remove('wide-layout');
    }

    // Clear previous content
    domCache.productionContainer.innerHTML = '';
    domCache.disasterContainer.innerHTML = '';
    domCache.lbSystemList.innerHTML = '';

    if (data.length > 0) {
        const ipLabel = window.I18N && window.I18N.lbIpLabel ? window.I18N.lbIpLabel : 'IP';
        const lbIp = (window.APP_CONFIG && window.APP_CONFIG.frontend && window.APP_CONFIG.frontend.load_balancer_ip) || 'N/A';
        domCache.lbIpValue.textContent = lbIp;
        domCache.lbIpLabel.textContent = ipLabel;
    } else {
        domCache.lbIpValue.textContent = 'N/A';
    }

    data.forEach(db => {
        domCache.productionContainer.appendChild(dbCardTemplate(db, 'production'));
        domCache.disasterContainer.appendChild(dbCardTemplate(db, 'disaster'));
        domCache.lbSystemList.appendChild(lbItemTemplate(db));
    });
}

function showError(message) {
    console.error(message);
    const errorMessage = `<div class="error-message">${t('dataLoadError')}: ${message}</div>`;
    domCache.productionContainer.innerHTML = errorMessage;
    domCache.disasterContainer.innerHTML = '';
    domCache.lbSystemList.innerHTML = '';
}

// --- Template Functions ---
function dbCardTemplate(db, type) {
    const template = document.getElementById('db-card-template').content.cloneNode(true);
    const card = template.querySelector('.db-card');

    const isProduction = type === 'production';
    const data = {
        ip: isProduction ? db.production_ip : db.disaster_ip,
        alive: isProduction ? db.production_alive : db.disaster_alive,
        portAlive: isProduction ? db.production_port_1521 : db.disaster_port_1521,
        dbConnect: isProduction ? db.production_db_connect : db.disaster_db_connect,
        status: isProduction ? (db.production_status || (db.production_alive ? 'OK' : 'Offline')) : (db.disaster_status || (db.disaster_alive ? 'OK' : 'Offline')),
        role: isProduction ? (db.production_role || 'Primary') : (db.disaster_role || 'Standby'),
        delay: isProduction ? null : db.disaster_dgdelay,
        connections: isProduction ? db.connections : null,
    };

    const targetEnv = determineLoadBalancerTarget(db);
    const isTargetOfLB = (isProduction && targetEnv === 'targetProd') || (!isProduction && targetEnv === 'targetDR');

    // --- Set Content ---
    card.querySelector('.db-name-text').textContent = db.name;
    card.querySelector('.ip').textContent = data.ip;
    card.querySelector('.role-item').innerHTML = `${t('roleLabel')}: ${t(data.role)}`;
    card.querySelector('.overall-status-text').textContent = t(data.status);

    // --- Set Status Classes ---
    card.querySelector('.ping-status').classList.add(data.alive ? 'status-online' : 'status-offline');
    card.querySelector('.port-status').classList.add(data.portAlive ? 'status-online' : 'status-offline');
    card.querySelector('.db-connect-status').classList.add(data.dbConnect ? 'status-online' : 'status-offline');

    let overallStatusClass = 'status-offline';
    if (data.alive && data.portAlive && data.dbConnect) {
        overallStatusClass = data.status === 'Warning' ? 'status-warning' : 'status-online';
    } else if (data.alive || data.portAlive) {
        overallStatusClass = 'status-warning';
    }
    card.querySelector('.overall-status').classList.add(overallStatusClass);

    // --- Conditional Rendering ---
    if (isTargetOfLB) {
        card.querySelector('.load-direction').style.display = 'flex';
    }

    if (isProduction && data.alive && db.disaster_alive) {
        const dataFlow = card.querySelector('.data-flow-indicator');
        dataFlow.style.display = 'flex';
        let connsClass = 'success-color';
        if (data.connections < 1) connsClass = 'error-color';
        dataFlow.querySelector('.connections-count').innerHTML = `${t('connectionsLabel')} <span style="color: var(--${connsClass})">${data.connections}</span>`;
    }

    if (!isProduction && data.alive) {
        const delayItem = card.querySelector('.delay-item');
        let delayClass = 'success-color';
        if (data.delay > 60) delayClass = 'error-color';
        else if (data.delay > 5) delayClass = 'warning-color';
        delayItem.innerHTML = `${t('delayLabel')}: <span style="color: var(--${delayClass})">${data.delay}s</span>`;
    } else {
        card.querySelector('.delay-item').innerHTML = '&nbsp;';
    }

    return card;
}


function lbItemTemplate(db) {
    const template = document.getElementById('lb-item-template').content.cloneNode(true);
    const item = template.querySelector('.lb-system');

    const targetEnv = determineLoadBalancerTarget(db);

    // --- Set Content ---
    item.querySelector('.lb-name').textContent = db.name.split('数据库')[0];
    item.querySelector('.lb-target-env').textContent = t(targetEnv);
    item.querySelector('.lb-ip-text').textContent = db.load_balancer_ip;

    // --- Set Status Classes ---
    item.querySelector('.ping-status').classList.add(db.load_balancer_alive ? 'status-online' : 'status-offline');
    item.querySelector('.port-status').classList.add(db.load_balancer_port_1521 ? 'status-online' : 'status-offline');
    item.querySelector('.db-connect-status').classList.add(db.load_balancer_db_connect ? 'status-online' : 'status-offline');

    // --- Set Background Color ---
    let bgColor = 'rgba(100, 100, 100, 0.3)';
    if (targetEnv === 'targetProd') {
        bgColor = 'rgba(24, 144, 255, 0.3)';
    } else if (targetEnv === 'targetDR') {
        bgColor = 'rgba(230, 100, 60, 0.3)';
    }
    item.style.backgroundColor = bgColor;

    return item;
}

// --- Helper Functions ---
function determineLoadBalancerTarget(db) {
    if (!db.load_balancer_alive) {
        return 'targetOffline';
    }
    if (db.production_alive && db.production_role === "PRIMARY") {
        return 'targetProd';
    } else if (db.disaster_alive && db.disaster_role === "PRIMARY") {
        return 'targetDR';
    } else {
        return 'targetOffline';
    }
}

function adjustGridForFitScreen(totalCards) {
    const screenW = window.innerWidth;
    const screenH = window.innerHeight - 120; // Reserve height for top title and buttons
    const minCardW = 240, minCardH = 150;
    let maxCols = Math.floor(screenW / minCardW);
    let maxRows = Math.floor(screenH / minCardH);
    maxCols = Math.max(1, maxCols);
    maxRows = Math.max(1, maxRows);
    let cols = Math.min(totalCards, maxCols);
    let rows = Math.ceil(totalCards / cols);
    while (rows > maxRows && cols < totalCards) {
        cols++;
        rows = Math.ceil(totalCards / cols);
    }
    document.documentElement.style.setProperty('--db-grid-columns', cols);
}

function fitAllCards() {
    const allCards = document.querySelectorAll('.db-card');
    if (allCards.length > 0) {
        adjustGridForFitScreen(allCards.length);
    }
}

window.addEventListener('resize', fitAllCards);
// --- Fullscreen Toggle ---
function toggleFullScreen() {
    if (!document.fullscreenElement) {
        document.documentElement.requestFullscreen().catch(err => {
            alert(`Error attempting to enable full-screen mode: ${err.message} (${err.name})`);
        });
    } else {
        if (document.exitFullscreen) {
            document.exitFullscreen();
        }
    }
}

function updateFullscreenButton() {
    if (document.fullscreenElement) {
        domCache.fullscreenBtn.textContent = t('exitFullscreen');
    } else {
        domCache.fullscreenBtn.textContent = t('fullscreen');
    }
}

window.addEventListener('resize', fitAllCards);
document.addEventListener('fullscreenchange', () => {
    fitAllCards();
    updateFullscreenButton();
});
window.addEventListener('DOMContentLoaded', fitAllCards);

// Initialization
async function init() {
    // Cache DOM elements
    domCache.mainTitleH1 = document.getElementById('main-title-h1');
    domCache.prodDcTitle = document.getElementById('prod-dc-title');
    domCache.drDcTitle = document.getElementById('dr-dc-title');
    domCache.productionContainer = document.getElementById('production-container');
    domCache.disasterContainer = document.getElementById('disaster-container');
    domCache.lbIpValue = document.querySelector('[data-i18n-target="lb-ip-value"]');
    domCache.lbIpLabel = document.querySelector('[data-i18n="lbIpLabel"]');
    domCache.currentTime = document.getElementById('current-time');
    domCache.lbSystemList = document.getElementById('lb-system-list');
        domCache.dashboardContainer = document.querySelector('.dashboard');
    domCache.fullscreenBtn = document.getElementById('fullscreen-btn');

    await loadTranslations(); // Load translations first

    applyTranslations();
    updateTitles(window.APP_TITLES);
    applyLayout();
    updateCurrentTime();
    setInterval(updateCurrentTime, 1000);

    fetchAndRenderData();

    // Add fullscreen button listener
    if (domCache.fullscreenBtn) {
        domCache.fullscreenBtn.addEventListener('click', toggleFullScreen);
        updateFullscreenButton(); // Set initial text
    }

    let refreshTimer;
    function setDynamicRefresh() {
        const currentHour = new Date().getHours();
        const frontendConfig = window.APP_CONFIG && window.APP_CONFIG.frontend;
        let interval = (frontendConfig && frontendConfig.default_interval_ms) || 600000; // Default to 10 minutes

        if (frontendConfig && frontendConfig.refresh_intervals) {
            for (const slot of frontendConfig.refresh_intervals) {
                if (currentHour >= slot.start_hour && currentHour < slot.end_hour) {
                    interval = slot.interval_ms;
                    break;
                }
            }
        }

        if (refreshTimer) clearTimeout(refreshTimer);

        refreshTimer = setTimeout(() => {
            fetchAndRenderData();
            setDynamicRefresh();
        }, interval);
    }
    setDynamicRefresh();

    window.addEventListener('visibilitychange', () => {
        if (!document.hidden) {
           fetchAndRenderData();
        }
    });
}

// Run init function when the DOM is fully loaded
document.addEventListener('DOMContentLoaded', init);
