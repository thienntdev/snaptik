/**
 * SnapTiktok - Frontend JavaScript
 * Minimal JS for form handling, result display, and local storage history
 */

(function () {
    'use strict';

    // ===== DOM Elements =====
    const form = document.getElementById('download-form');
    const urlInput = document.getElementById('url-input');
    const submitBtn = document.getElementById('submit-btn');
    const btnText = document.getElementById('btn-text');
    const btnIconDefault = document.getElementById('btn-icon-default');
    const btnSpinner = document.getElementById('btn-spinner');
    const errorMessage = document.getElementById('error-message');
    const resultSection = document.getElementById('result-section');
    const pasteBtn = document.getElementById('paste-btn');

    // Result elements
    const previewThumbnail = document.getElementById('preview-thumbnail');
    const previewVideo = document.getElementById('preview-video');
    const playOverlay = document.getElementById('play-overlay');
    const resultTitle = document.getElementById('result-title');
    const resultAuthor = document.getElementById('result-author');
    const resultPlatform = document.getElementById('result-platform');
    const resultAvatar = document.getElementById('result-avatar');

    // Stats
    const statViews = document.getElementById('stat-views');
    const statLikes = document.getElementById('stat-likes');
    const statComments = document.getElementById('stat-comments');
    const statMusic = document.getElementById('stat-music');

    // Download buttons
    const btnDownloadVideoHD = document.getElementById('btn-download-video-hd');
    const btnDownloadVideo = document.getElementById('btn-download-video');
    const btnDownloadAudio = document.getElementById('btn-download-audio');
    const btnDownloadImagesContainer = document.getElementById('btn-download-images-container');
    const imagesGrid = document.getElementById('images-grid');
    const btnCopyLink = document.getElementById('btn-copy-link');

    // Ad slots
    const adAboveResult = document.getElementById('ad-above-result');

    // History
    const historySection = document.getElementById('history-section');
    const historyList = document.getElementById('history-list');
    const clearHistoryBtn = document.getElementById('clear-history');

    // State
    let currentVideoData = null;

    // ===== URL Validation =====
    function isValidURL(str) {
        try {
            const url = new URL(str);
            const host = url.hostname.toLowerCase();
            const validHosts = [
                'tiktok.com', 'www.tiktok.com', 'vm.tiktok.com', 'vt.tiktok.com', 'm.tiktok.com',
                'douyin.com', 'www.douyin.com', 'v.douyin.com', 'm.douyin.com', 'www.iesdouyin.com'
            ];
            return validHosts.some(h => host === h || host.endsWith('.' + h));
        } catch {
            return false;
        }
    }

    // ===== Form Submission =====
    if (form) {
        form.addEventListener('submit', async function (e) {
            e.preventDefault();
            const url = urlInput.value.trim();

            // Validate
            if (!url) {
                showError('Please paste a TikTok or Douyin URL.');
                return;
            }

            // Add protocol if missing
            let normalizedUrl = url;
            if (!url.startsWith('http://') && !url.startsWith('https://')) {
                normalizedUrl = 'https://' + url;
            }

            if (!isValidURL(normalizedUrl)) {
                showError('Please enter a valid TikTok or Douyin URL.');
                return;
            }

            // Start loading
            setLoading(true);
            hideError();
            hideResult();

            try {
                const response = await fetch('/api/parse', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: normalizedUrl })
                });

                const data = await response.json();

                if (!data.success) {
                    showError(data.error || 'Failed to process video. Please try again.');
                    return;
                }

                // Display result
                currentVideoData = data.data;
                displayResult(data.data);
                saveToHistory(data.data);

            } catch (err) {
                console.error('Parse error:', err);
                showError('Network error. Please check your connection and try again.');
            } finally {
                setLoading(false);
            }
        });
    }

    // ===== Display Result =====
    function displayResult(data) {
        if (!resultSection) return;

        // Thumbnail
        if (data.cover_url && previewThumbnail) {
            previewThumbnail.src = data.cover_url;
            previewThumbnail.alt = data.title || 'Video thumbnail';
            previewThumbnail.classList.remove('hidden');
            if (previewVideo) previewVideo.classList.add('hidden');
            if (playOverlay) playOverlay.classList.remove('hidden');
        }

        // Info
        if (resultTitle) resultTitle.textContent = data.title || 'Untitled';
        if (resultAuthor) resultAuthor.textContent = data.author || 'Unknown';
        if (resultPlatform) {
            resultPlatform.textContent = data.platform === 'douyin' ? '抖音 Douyin' : 'TikTok';
        }

        // Avatar
        if (data.author_avatar && resultAvatar) {
            resultAvatar.src = data.author_avatar;
            resultAvatar.classList.remove('hidden');
        }

        // Stats
        if (data.views && statViews) {
            statViews.querySelector('span:last-child').textContent = formatNumber(data.views);
            statViews.classList.remove('hidden');
        }
        if (data.likes && statLikes) {
            statLikes.querySelector('span:last-child').textContent = formatNumber(data.likes);
            statLikes.classList.remove('hidden');
        }
        if (data.comments && statComments) {
            statComments.querySelector('span:last-child').textContent = formatNumber(data.comments);
            statComments.classList.remove('hidden');
        }
        if (data.music && statMusic) {
            statMusic.querySelector('span:last-child').textContent = data.music;
            statMusic.classList.remove('hidden');
        }

        // Download buttons
        const filename = `snaptiktok_${data.id || 'video'}`;

        if (data.video_hd_url && btnDownloadVideoHD) {
            btnDownloadVideoHD.href = `/api/download?url=${encodeURIComponent(data.video_hd_url)}&type=video&filename=${filename}_hd`;
            btnDownloadVideoHD.classList.remove('hidden');
        }

        if (data.video_url && btnDownloadVideo) {
            btnDownloadVideo.href = `/api/download?url=${encodeURIComponent(data.video_url)}&type=video&filename=${filename}`;
            btnDownloadVideo.classList.remove('hidden');
        }

        if (data.audio_url && btnDownloadAudio) {
            btnDownloadAudio.href = `/api/download?url=${encodeURIComponent(data.audio_url)}&type=audio&filename=${filename}_audio`;
            btnDownloadAudio.classList.remove('hidden');
        }

        // Images (slideshow posts)
        if (data.images && data.images.length > 0 && btnDownloadImagesContainer && imagesGrid) {
            imagesGrid.innerHTML = '';
            data.images.forEach((imgUrl, index) => {
                const link = document.createElement('a');
                link.href = `/api/download?url=${encodeURIComponent(imgUrl)}&type=image&filename=${filename}_img${index + 1}`;
                link.className = 'block rounded-lg overflow-hidden border border-white/10 hover:border-brand-500/30 transition-colors';
                link.download = true;

                const img = document.createElement('img');
                img.src = imgUrl;
                img.alt = `Image ${index + 1}`;
                img.className = 'w-full h-20 object-cover';
                img.loading = 'lazy';

                link.appendChild(img);
                imagesGrid.appendChild(link);
            });
            btnDownloadImagesContainer.classList.remove('hidden');
        }

        // Show result section
        resultSection.classList.remove('hidden');
        resultSection.scrollIntoView({ behavior: 'smooth', block: 'start' });

        // Show ad above result
        if (adAboveResult) {
            adAboveResult.classList.remove('hidden');
        }

        // Show mobile sticky ad after a delay
        setTimeout(() => {
            const mobileAd = document.getElementById('sticky-ad-mobile');
            if (mobileAd) mobileAd.classList.remove('hidden');
        }, 2000);
    }

    // ===== Video Play Overlay =====
    if (playOverlay) {
        playOverlay.addEventListener('click', function () {
            if (!currentVideoData) return;

            const videoUrl = currentVideoData.video_url || currentVideoData.video_hd_url;
            if (!videoUrl || !previewVideo || !previewThumbnail) return;

            // Use proxy URL to avoid CORS
            previewVideo.src = `/api/download?url=${encodeURIComponent(videoUrl)}&type=video&filename=preview`;
            previewVideo.classList.remove('hidden');
            previewThumbnail.classList.add('hidden');
            playOverlay.classList.add('hidden');
            previewVideo.play().catch(() => { /* autoplay blocked is fine */ });
        });
    }

    // ===== Copy Link =====
    if (btnCopyLink) {
        btnCopyLink.addEventListener('click', function () {
            if (!currentVideoData) return;

            const url = currentVideoData.video_url || currentVideoData.video_hd_url || currentVideoData.original_url;
            if (!url) return;

            navigator.clipboard.writeText(url).then(() => {
                showToast('Link copied to clipboard!');
                btnCopyLink.querySelector('span').textContent = 'Copied!';
                setTimeout(() => {
                    btnCopyLink.querySelector('span').textContent = 'Copy Link';
                }, 2000);
            }).catch(() => {
                showToast('Failed to copy');
            });
        });
    }

    // ===== Paste Button (Mobile) =====
    if (pasteBtn) {
        pasteBtn.addEventListener('click', async function () {
            try {
                const text = await navigator.clipboard.readText();
                if (text && urlInput) {
                    urlInput.value = text;
                    urlInput.focus();
                }
            } catch {
                showToast('Unable to paste. Please paste manually.');
            }
        });
    }

    // ===== Loading State =====
    function setLoading(loading) {
        if (!submitBtn) return;

        submitBtn.disabled = loading;
        if (btnIconDefault) btnIconDefault.classList.toggle('hidden', loading);
        if (btnSpinner) btnSpinner.classList.toggle('hidden', !loading);
        if (btnText) btnText.textContent = loading ? 'Processing...' : (document.querySelector('[id="btn-text"]')?.dataset?.defaultText || 'Download Now');
    }

    // ===== Error Handling =====
    function showError(message) {
        if (!errorMessage) return;
        errorMessage.textContent = message;
        errorMessage.classList.remove('hidden');
    }

    function hideError() {
        if (errorMessage) errorMessage.classList.add('hidden');
    }

    // ===== Hide Result =====
    function hideResult() {
        if (!resultSection) return;
        resultSection.classList.add('hidden');

        // Reset buttons
        [btnDownloadVideoHD, btnDownloadVideo, btnDownloadAudio].forEach(btn => {
            if (btn) btn.classList.add('hidden');
        });
        if (btnDownloadImagesContainer) btnDownloadImagesContainer.classList.add('hidden');

        // Reset stats
        [statViews, statLikes, statComments, statMusic].forEach(stat => {
            if (stat) stat.classList.add('hidden');
        });

        if (resultAvatar) resultAvatar.classList.add('hidden');

        // Reset video
        if (previewVideo) {
            previewVideo.pause();
            previewVideo.src = '';
            previewVideo.classList.add('hidden');
        }
        if (previewThumbnail) previewThumbnail.classList.remove('hidden');
        if (playOverlay) playOverlay.classList.remove('hidden');
    }

    // ===== Format Number =====
    function formatNumber(num) {
        if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
        if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
        return num.toString();
    }

    // ===== Toast Notification =====
    function showToast(message) {
        // Remove existing toast
        const existing = document.querySelector('.toast');
        if (existing) existing.remove();

        const toast = document.createElement('div');
        toast.className = 'toast';
        toast.textContent = message;
        document.body.appendChild(toast);

        requestAnimationFrame(() => {
            toast.classList.add('show');
        });

        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, 2500);
    }

    // ===== Local Storage History =====
    const HISTORY_KEY = 'snaptiktok_history';
    const MAX_HISTORY = 12;

    function saveToHistory(data) {
        try {
            const history = getHistory();
            // Prevent duplicates
            const filtered = history.filter(item => item.id !== data.id);
            filtered.unshift({
                id: data.id,
                title: data.title,
                author: data.author,
                cover_url: data.cover_url,
                platform: data.platform,
                original_url: data.original_url,
                timestamp: Date.now()
            });
            // Keep only recent items
            localStorage.setItem(HISTORY_KEY, JSON.stringify(filtered.slice(0, MAX_HISTORY)));
            renderHistory();
        } catch (e) {
            // localStorage might be unavailable
        }
    }

    function getHistory() {
        try {
            const data = localStorage.getItem(HISTORY_KEY);
            return data ? JSON.parse(data) : [];
        } catch {
            return [];
        }
    }

    function renderHistory() {
        const history = getHistory();
        if (!historySection || !historyList || history.length === 0) return;

        historySection.classList.remove('hidden');
        historyList.innerHTML = '';

        history.forEach(item => {
            const card = document.createElement('div');
            card.className = 'group cursor-pointer rounded-xl overflow-hidden border border-white/5 hover:border-brand-500/20 transition-all duration-300 bg-dark-800/30 hover:bg-dark-800/50';
            card.onclick = () => {
                if (urlInput) {
                    urlInput.value = item.original_url;
                    urlInput.focus();
                    window.scrollTo({ top: 0, behavior: 'smooth' });
                }
            };

            card.innerHTML = `
                <div class="aspect-[9/16] max-h-32 bg-dark-900 overflow-hidden">
                    ${item.cover_url
                        ? `<img src="${item.cover_url}" alt="${item.title || 'Video'}" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" loading="lazy">`
                        : '<div class="w-full h-full flex items-center justify-center text-dark-600"><svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg></div>'
                    }
                </div>
                <div class="p-2.5">
                    <p class="text-xs text-dark-300 truncate">${item.title || 'Video'}</p>
                    <p class="text-xs text-dark-500 mt-0.5">@${item.author || 'Unknown'}</p>
                </div>
            `;

            historyList.appendChild(card);
        });
    }

    if (clearHistoryBtn) {
        clearHistoryBtn.addEventListener('click', () => {
            localStorage.removeItem(HISTORY_KEY);
            if (historySection) historySection.classList.add('hidden');
            showToast('History cleared');
        });
    }

    // ===== FAQ Accordion =====
    window.toggleFaq = function (btn) {
        const item = btn.closest('.faq-item');
        if (!item) return;

        const wasActive = item.classList.contains('active');

        // Close all FAQ items
        document.querySelectorAll('.faq-item').forEach(faq => {
            faq.classList.remove('active');
            const answer = faq.querySelector('.faq-answer');
            if (answer) {
                answer.style.maxHeight = null;
                answer.classList.add('hidden');
            }
        });

        // Toggle current
        if (!wasActive) {
            item.classList.add('active');
            const answer = item.querySelector('.faq-answer');
            if (answer) {
                answer.classList.remove('hidden');
                answer.style.maxHeight = answer.scrollHeight + 'px';
            }
        }
    };

    // ===== Init =====
    renderHistory();

    // Auto-focus input on desktop
    if (urlInput && window.innerWidth > 768) {
        urlInput.focus();
    }

})();
