#!/usr/bin/env bash
set -euo pipefail

echo "🌐 Testing Complete Multi-File Website Workflow"
echo "=============================================="
echo ""

# Clean up any existing test data
rm -rf /tmp/test-node /tmp/test-wallet.json /tmp/test-website

echo "1️⃣ Starting test node..."
./bin/betanet-node run -data /tmp/test-node -listen /ip4/0.0.0.0/tcp/4002 &
NODE_PID=$!
sleep 3

echo "2️⃣ Creating test wallet..."
./bin/betanet-wallet new -out /tmp/test-wallet.json > /tmp/wallet-output.txt
MNEMONIC=$(grep -A 1 "Mnemonic" /tmp/wallet-output.txt | tail -n 1 | xargs)
echo "   Wallet created with mnemonic (first 20 chars): ${MNEMONIC:0:20}..."

echo "3️⃣ Adding test site..."
./bin/betanet-wallet add-site -wallet /tmp/test-wallet.json -mnemonic "$MNEMONIC" -label testsite

echo "4️⃣ Creating multi-file test website..."
mkdir -p /tmp/test-website
mkdir -p /tmp/test-website/css
mkdir -p /tmp/test-website/js
mkdir -p /tmp/test-website/images

# Create index.html with references to CSS, JS, and images
cat > /tmp/test-website/index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Betanet Multi-File Website Test</title>
    <link rel="stylesheet" href="css/styles.css">
    <link rel="icon" href="images/favicon.ico" type="image/x-icon">
</head>
<body>
    <div class="container">
        <header class="header">
            <h1 class="title">🌐 Betanet Multi-File Website</h1>
            <p class="subtitle">Testing Complete Multi-File Functionality</p>
        </header>
        
        <main class="main">
            <section class="hero">
                <h2>Welcome to the Future of Decentralized Web!</h2>
                <p>This website demonstrates the full multi-file capabilities of Betanet:</p>
                <ul class="features">
                    <li>✅ HTML files with proper structure</li>
                    <li>✅ CSS styling and animations</li>
                    <li>✅ JavaScript interactivity</li>
                    <li>✅ Image assets and icons</li>
                    <li>✅ All files stored as separate blockchain transactions</li>
                </ul>
            </section>
            
            <section class="demo">
                <h3>Interactive Demo</h3>
                <div class="counter-container">
                    <p>Counter: <span id="counter">0</span></p>
                    <button id="increment" class="btn btn-primary">Increment</button>
                    <button id="decrement" class="btn btn-secondary">Decrement</button>
                    <button id="reset" class="btn btn-danger">Reset</button>
                </div>
                
                <div class="color-changer">
                    <p>Theme: <span id="current-theme">Light</span></p>
                    <button id="toggle-theme" class="btn btn-accent">Toggle Theme</button>
                </div>
            </section>
            
            <section class="info">
                <h3>Website Information</h3>
                <div class="info-grid">
                    <div class="info-item">
                        <strong>Domain:</strong> test.bn
                    </div>
                    <div class="info-item">
                        <strong>Site ID:</strong> <span id="site-id">Loading...</span>
                    </div>
                    <div class="info-item">
                        <strong>Files:</strong> <span id="file-count">Loading...</span>
                    </div>
                    <div class="info-item">
                        <strong>Last Updated:</strong> <span id="last-updated">Loading...</span>
                    </div>
                </div>
            </section>
        </main>
        
        <footer class="footer">
            <p>&copy; 2024 Betanet - Decentralized Web Platform</p>
            <p>Built with ❤️ using multi-file website technology</p>
        </footer>
    </div>
    
    <script src="js/app.js"></script>
</body>
</html>
EOF

# Create CSS file with modern styling and animations
cat > /tmp/test-website/css/styles.css << 'EOF'
/* Reset and base styles */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    line-height: 1.6;
    color: #333;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    transition: all 0.3s ease;
}

body.dark-theme {
    background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
    color: #ecf0f1;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
}

/* Header styles */
.header {
    text-align: center;
    padding: 40px 0;
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    border-radius: 20px;
    margin-bottom: 40px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.title {
    font-size: 3rem;
    margin-bottom: 10px;
    background: linear-gradient(45deg, #ff6b6b, #4ecdc4);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    animation: titleGlow 2s ease-in-out infinite alternate;
}

@keyframes titleGlow {
    from { filter: drop-shadow(0 0 10px rgba(255, 107, 107, 0.5)); }
    to { filter: drop-shadow(0 0 20px rgba(78, 205, 196, 0.5)); }
}

.subtitle {
    font-size: 1.2rem;
    color: #ecf0f1;
    opacity: 0.9;
}

/* Main content */
.main {
    display: grid;
    gap: 40px;
}

section {
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    border-radius: 15px;
    padding: 30px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
    transition: transform 0.3s ease, box-shadow 0.3s ease;
}

section:hover {
    transform: translateY(-5px);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);
}

section h2, section h3 {
    margin-bottom: 20px;
    color: #2c3e50;
    border-bottom: 2px solid #3498db;
    padding-bottom: 10px;
}

.dark-theme section h2, .dark-theme section h3 {
    color: #ecf0f1;
    border-bottom-color: #e74c3c;
}

/* Features list */
.features {
    list-style: none;
    padding: 0;
}

.features li {
    padding: 10px 0;
    border-left: 3px solid #27ae60;
    padding-left: 20px;
    margin: 10px 0;
    transition: all 0.3s ease;
}

.features li:hover {
    border-left-color: #2ecc71;
    transform: translateX(10px);
}

/* Demo section */
.counter-container, .color-changer {
    margin: 20px 0;
    padding: 20px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
}

/* Buttons */
.btn {
    padding: 12px 24px;
    margin: 5px;
    border: none;
    border-radius: 25px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    text-transform: uppercase;
    letter-spacing: 1px;
}

.btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(0, 0, 0, 0.2);
}

.btn-primary {
    background: linear-gradient(45deg, #3498db, #2980b9);
    color: white;
}

.btn-secondary {
    background: linear-gradient(45deg, #95a5a6, #7f8c8d);
    color: white;
}

.btn-danger {
    background: linear-gradient(45deg, #e74c3c, #c0392b);
    color: white;
}

.btn-accent {
    background: linear-gradient(45deg, #f39c12, #e67e22);
    color: white;
}

/* Info grid */
.info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    margin-top: 20px;
}

.info-item {
    padding: 15px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    border-left: 4px solid #3498db;
}

/* Footer */
.footer {
    text-align: center;
    padding: 30px 0;
    margin-top: 40px;
    border-top: 1px solid rgba(255, 255, 255, 0.2);
}

.footer p {
    margin: 5px 0;
    opacity: 0.8;
}

/* Responsive design */
@media (max-width: 768px) {
    .title {
        font-size: 2rem;
    }
    
    .container {
        padding: 10px;
    }
    
    section {
        padding: 20px;
    }
    
    .info-grid {
        grid-template-columns: 1fr;
    }
}

/* Loading animation */
.loading {
    opacity: 0.6;
    animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
}
EOF

# Create JavaScript file with interactive functionality
cat > /tmp/test-website/js/app.js << 'EOF'
// Main application JavaScript
class BetanetWebsite {
    constructor() {
        this.counter = 0;
        this.isDarkTheme = false;
        this.init();
    }
    
    init() {
        this.setupEventListeners();
        this.loadWebsiteInfo();
        this.setupAnimations();
        console.log('🌐 Betanet Multi-File Website loaded successfully!');
    }
    
    setupEventListeners() {
        // Counter functionality
        document.getElementById('increment').addEventListener('click', () => {
            this.counter++;
            this.updateCounter();
            this.showNotification('Counter incremented!', 'success');
        });
        
        document.getElementById('decrement').addEventListener('click', () => {
            this.counter--;
            this.updateCounter();
            this.showNotification('Counter decremented!', 'info');
        });
        
        document.getElementById('reset').addEventListener('click', () => {
            this.counter = 0;
            this.updateCounter();
            this.showNotification('Counter reset!', 'warning');
        });
        
        // Theme toggle
        document.getElementById('toggle-theme').addEventListener('click', () => {
            this.toggleTheme();
        });
        
        // Add some fun interactions
        this.addHoverEffects();
    }
    
    updateCounter() {
        const counterElement = document.getElementById('counter');
        counterElement.textContent = this.counter;
        
        // Add animation
        counterElement.style.transform = 'scale(1.2)';
        setTimeout(() => {
            counterElement.style.transform = 'scale(1)';
        }, 200);
        
        // Store in localStorage
        localStorage.setItem('betanet-counter', this.counter);
    }
    
    toggleTheme() {
        this.isDarkTheme = !this.isDarkTheme;
        document.body.classList.toggle('dark-theme', this.isDarkTheme);
        
        const themeText = document.getElementById('current-theme');
        themeText.textContent = this.isDarkTheme ? 'Dark' : 'Light';
        
        // Store preference
        localStorage.setItem('betanet-theme', this.isDarkTheme);
        
        this.showNotification(
            `Theme changed to ${this.isDarkTheme ? 'Dark' : 'Light'} mode!`, 
            'success'
        );
    }
    
    setupAnimations() {
        // Add entrance animations
        const sections = document.querySelectorAll('section');
        sections.forEach((section, index) => {
            section.style.opacity = '0';
            section.style.transform = 'translateY(30px)';
            
            setTimeout(() => {
                section.style.transition = 'all 0.6s ease';
                section.style.opacity = '1';
                section.style.transform = 'translateY(0)';
            }, index * 200);
        });
        
        // Add floating animation to title
        const title = document.querySelector('.title');
        title.style.animation = 'titleGlow 2s ease-in-out infinite alternate, float 3s ease-in-out infinite';
        
        // Add floating keyframe
        if (!document.querySelector('#floating-keyframes')) {
            const style = document.createElement('style');
            style.id = 'floating-keyframes';
            style.textContent = `
                @keyframes float {
                    0%, 100% { transform: translateY(0px); }
                    50% { transform: translateY(-10px); }
                }
            `;
            document.head.appendChild(style);
        }
    }
    
    addHoverEffects() {
        // Add ripple effect to buttons
        const buttons = document.querySelectorAll('.btn');
        buttons.forEach(button => {
            button.addEventListener('mouseenter', (e) => {
                this.createRipple(e, button);
            });
        });
        
        // Add particle effect to header
        const header = document.querySelector('.header');
        header.addEventListener('mousemove', (e) => {
            this.createParticle(e, header);
        });
    }
    
    createRipple(event, button) {
        const ripple = document.createElement('span');
        const rect = button.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        const x = event.clientX - rect.left - size / 2;
        const y = event.clientY - rect.top - size / 2;
        
        ripple.style.width = ripple.style.height = size + 'px';
        ripple.style.left = x + 'px';
        ripple.style.top = y + 'px';
        ripple.classList.add('ripple');
        
        button.appendChild(ripple);
        
        setTimeout(() => {
            ripple.remove();
        }, 600);
    }
    
    createParticle(event, element) {
        if (Math.random() > 0.1) return; // Limit particle creation
        
        const particle = document.createElement('div');
        particle.style.position = 'absolute';
        particle.style.width = '4px';
        particle.style.height = '4px';
        particle.style.background = '#fff';
        particle.style.borderRadius = '50%';
        particle.style.pointerEvents = 'none';
        particle.style.left = (event.clientX - element.getBoundingClientRect().left) + 'px';
        particle.style.top = (event.clientY - element.getBoundingClientRect().top) + 'px';
        
        element.appendChild(particle);
        
        const animation = particle.animate([
            { transform: 'translateY(0px)', opacity: 1 },
            { transform: 'translateY(-20px)', opacity: 0 }
        ], {
            duration: 1000,
            easing: 'ease-out'
        });
        
        animation.onfinish = () => particle.remove();
    }
    
    async loadWebsiteInfo() {
        try {
            // Simulate loading website information
            // In a real scenario, this would fetch from the blockchain
            await this.simulateLoading();
            
            // Update the display with mock data (replace with real data)
            document.getElementById('site-id').textContent = 'test-site-123';
            document.getElementById('file-count').textContent = '5 files';
            document.getElementById('last-updated').textContent = new Date().toLocaleString();
            
            // Remove loading state
            document.querySelectorAll('.loading').forEach(el => {
                el.classList.remove('loading');
            });
            
        } catch (error) {
            console.error('Failed to load website info:', error);
            this.showNotification('Failed to load website info', 'error');
        }
    }
    
    simulateLoading() {
        return new Promise(resolve => {
            setTimeout(resolve, 1500);
        });
    }
    
    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        // Style the notification
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 15px 20px;
            border-radius: 8px;
            color: white;
            font-weight: 600;
            z-index: 1000;
            transform: translateX(100%);
            transition: transform 0.3s ease;
            box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        `;
        
        // Set background color based on type
        const colors = {
            success: '#27ae60',
            error: '#e74c3c',
            warning: '#f39c12',
            info: '#3498db'
        };
        notification.style.background = colors[type] || colors.info;
        
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.style.transform = 'translateX(0)';
        }, 100);
        
        // Animate out and remove
        setTimeout(() => {
            notification.style.transform = 'translateX(100%)';
            setTimeout(() => {
                notification.remove();
            }, 300);
        }, 3000);
    }
}

// Initialize the website when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.betanetWebsite = new BetanetWebsite();
    
    // Load saved preferences
    const savedCounter = localStorage.getItem('betanet-counter');
    if (savedCounter !== null) {
        window.betanetWebsite.counter = parseInt(savedCounter);
        window.betanetWebsite.updateCounter();
    }
    
    const savedTheme = localStorage.getItem('betanet-theme');
    if (savedTheme === 'true') {
        window.betanetWebsite.toggleTheme();
    }
});

// Add some global utility functions
window.betanetUtils = {
    getRandomColor: () => {
        const colors = ['#3498db', '#e74c3c', '#2ecc71', '#f39c12', '#9b59b6'];
        return colors[Math.floor(Math.random() * colors.length)];
    },
    
    animateElement: (element, animation) => {
        element.style.animation = animation;
        setTimeout(() => {
            element.style.animation = '';
        }, 1000);
    }
};
EOF

# Create a simple favicon
echo "Creating favicon placeholder..."
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" | base64 -d > /tmp/test-website/images/favicon.ico

echo "5️⃣ Publishing multi-file website..."
./bin/betanet-wallet publish-website \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -dir /tmp/test-website \
  -main index.html \
  -data /tmp/test-wallet-db

echo "6️⃣ Registering domain 'test.bn'..."
./bin/betanet-wallet register-domain \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -domain test.bn \
  -data /tmp/test-wallet-db

echo "7️⃣ Verifying domain resolution..."
./bin/betanet-wallet resolve-domain -data /tmp/test-wallet-db -domain test.bn

echo "8️⃣ Stopping test node and publishing website to node database..."
# Stop the test node so we can access its database
kill $NODE_PID || true
sleep 2

# Use the wallet to publish the website to the node database
./bin/betanet-wallet publish-website \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -dir /tmp/test-website \
  -main index.html \
  -data /tmp/test-node

echo "8️⃣ Registering domain in node database..."
# Register the domain directly in the node database
./bin/betanet-wallet register-domain \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -domain test.bn \
  -data /tmp/test-node

echo "9️⃣ Listing website contents..."
./bin/betanet-wallet list-website \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -data /tmp/test-wallet-db

echo "🔟 Getting detailed website information..."
./bin/betanet-wallet get-website-info \
  -wallet /tmp/test-wallet.json \
  -mnemonic "$MNEMONIC" \
  -label testsite \
  -data /tmp/test-wallet-db

echo ""
echo "✅ Multi-file website workflow complete!"
echo ""
echo "🌐 Your multi-file website is now live at 'test.bn'!"
echo ""
echo "📁 Website files created:"
echo "   - /tmp/test-website/index.html (main HTML file)"
echo "   - /tmp/test-website/css/styles.css (styling and animations)"
echo "   - /tmp/test-website/js/app.js (interactive functionality)"
echo "   - /tmp/test-website/images/favicon.ico (website icon)"
echo ""
echo "🚀 Test the website by opening the browser:"
echo "   ./bin/betanet-browser -data /tmp/test-node"
echo ""
echo "   Then navigate to 'test.bn' to see your full multi-file website!"
echo ""
echo "✨ Features to test:"
echo "   - Responsive design and animations"
echo "   - Interactive counter with buttons"
echo "   - Dark/light theme toggle"
echo "   - Hover effects and particle animations"
echo "   - Local storage for preferences"
echo "   - Full CSS styling and JavaScript functionality"
echo ""
# echo "🧹 Cleanup: kill $NODE_PID && rm -rf /tmp/test-node /tmp/test-wallet.json /tmp/test-website"
