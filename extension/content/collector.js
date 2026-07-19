// content/collector.js — 商品页采集 + 行为采集（注入到 1688/阿里巴巴/亚马逊页面）。
//
// 采集策略：
//   1. 根据 URL 判断平台
//   2. 用多组 selector fallback 提取商品字段（容错 DOM 变化）
//   3. 行为采集：记录浏览时长（visibilitychange 上报）
//
// 与 background 的消息协议：
//   { type: 'COLLECT_PAGE' }   → 采集当前页并上报
//   { type: 'GET_PAGE_INFO' }  → 返回当前页可采集信息（popup 显示用）

(() => {
  const PLATFORM = detectPlatform(location.href);
  if (!PLATFORM) return; // 非采集目标站

  // ===== 平台检测 =====
  function detectPlatform(url) {
    const u = url.toLowerCase();
    if (u.includes('1688.com')) return '1688';
    if (u.includes('alibaba.com')) return 'alibaba';
    if (u.includes('amazon.')) return 'amazon';
    return null;
  }

  // ===== 通用提取工具：按多 selector 尝试 =====
  function pick(selectors, { attr, all } = {}) {
    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el) {
        if (attr) return el.getAttribute(attr) || '';
        return el.textContent.trim();
      }
    }
    if (all) {
      for (const sel of selectors) {
        const els = document.querySelectorAll(sel);
        if (els.length) return Array.from(els).map(e => e.textContent.trim());
      }
    }
    return '';
  }

  function pickText(selectors) { return pick(selectors); }
  function pickImg(selectors) { return pick(selectors, { attr: 'src' }); }

  // ===== 各平台采集器 =====
  const COLLECTORS = {
    '1688': () => {
      const name = pickText([
        '.title-text', '.mod-detail-title',
        'h1.title', '[class*="offer-title"]', 'meta[property="og:title"]',
      ]) || document.title;
      const priceText = pickText([
        '.price-content .value', '.value.price',
        '[class*="price"] .num', 'meta[property="og:product:price:amount"]',
      ]);
      const images = (document.querySelectorAll(
        '.detail-gallery img, .tab-trigger img, [class*="gallery"] img'
      ));
      const imgURLs = Array.from(images)
        .map(i => i.src || i.getAttribute('data-src'))
        .filter(s => s && s.startsWith('http'))
        .slice(0, 8);
      const moqText = pickText(['.unit.player.moq .value', '[class*="moq"]']);
      const supplier = pickText([
        '.company-name .name', '.company-name',
        '[class*="supplier"] [class*="name"]', '.shop-name',
      ]);
      const category = pickText(['.crumb a:last-child', '.breadcrumb a:last-child']);
      const sourceId = new URLSearchParams(location.search).get('offerid') ||
                       location.pathname.match(/\d{10,}/)?.[0] || location.href;
      return {
        source: '1688', source_id: String(sourceId), source_url: location.href,
        name, description: name,
        price: extractNumber(priceText), price_currency: 'CNY',
        moq: extractNumber(moqText) || undefined,
        image_urls: imgURLs, category,
        supplier: { name: supplier, source_id: sourceId, location: '中国' },
      };
    },
    'alibaba': () => {
      const name = pickText([
        '.module-pdp-title h1', '[class*="product-title"]',
        'h1', 'meta[property="og:title"]',
      ]) || document.title;
      const priceText = pickText([
        '.pre-inquiry-price', '[class*="price"] .value',
        '.pdp-price', 'meta[property="og:product:price:amount"]',
      ]);
      const moqText = pickText(['.moq .value', '[class*="moq"]']);
      const supplier = pickText([
        '.company-name a', '[class*="supplier"] a',
        '.contact-info .company',
      ]);
      const images = document.querySelectorAll(
        '[class*="gallery"] img, .detail-main img, [class*="image-layout"] img'
      );
      const imgURLs = Array.from(images)
        .map(i => i.src).filter(s => s && s.startsWith('http')).slice(0, 8);
      const sourceId = location.pathname.match(/\/(\d{6,})\.html/)?.[1] ||
                       new URLSearchParams(location.search).get('spm') || location.href;
      return {
        source: 'alibaba', source_id: String(sourceId), source_url: location.href,
        name, description: name,
        price: extractNumber(priceText), price_currency: 'USD',
        moq: extractNumber(moqText) || undefined,
        image_urls: imgURLs,
        supplier: { name: supplier, source_id: sourceId, location: '' },
      };
    },
    'amazon': () => {
      const name = pickText(['#productTitle', '#title']) || document.title;
      const priceText = pickText([
        '#priceblock_ourprice', '#priceblock_saleprice',
        '.a-price .a-offscreen', '#corePrice_feature_div .a-offscreen',
      ]);
      const asin = location.pathname.match(/\/dp\/([A-Z0-9]{10})/)?.[1] ||
                   new URLSearchParams(location.search).get('asin') || '';
      const images = document.querySelectorAll(
        '#imgTagWrapperId img, #landingImage, #altImages img'
      );
      const imgURLs = Array.from(images)
        .map(i => (i.src || i.getAttribute('data-old-hires') || ''))
        .filter(s => s.startsWith('http')).slice(0, 8);
      const seller = pickText([
        '#sellerProfileTriggerId', '#bylineInfo',
      ]);
      return {
        source: 'amazon', source_id: asin, source_url: location.href,
        name, description: name,
        price: extractNumber(priceText), price_currency: 'USD',
        image_urls: imgURLs,
        supplier: { name: seller.replace(/^by\s+/i, ''), source_id: asin, location: '' },
        scenarios: ['b2c'],
      };
    },
  };

  function extractNumber(text) {
    if (!text) return '';
    const m = text.replace(/,/g, '').match(/[\d.]+/);
    return m ? m[0] : '';
  }

  // ===== 采集当前页 =====
  function collect() {
    const fn = COLLECTORS[PLATFORM];
    if (!fn) return null;
    try {
      const data = fn();
      return data?.name ? data : null;
    } catch (e) {
      console.warn('[TradeMind] 采集失败:', e);
      return null;
    }
  }

  // ===== 行为采集：浏览时长 =====
  let enterTime = Date.now();
  let reported = false;

  function reportBrowse(durationSec) {
    if (reported) return;
    reported = true;
    const data = collect();
    chrome.runtime?.sendMessage({
      type: 'BEHAVIOR',
      event: {
        event_type: 'browse',
        source: PLATFORM,
        target_id: data?.source_id || '',
        target_meta: data ? { name: data.name, price: data.price, url: location.href } : { url: location.href },
        duration_sec: durationSec,
        occurred_at: new Date().toISOString(),
      },
    }).catch(() => {});
  }

  // 页面隐藏 / 切走时上报
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') {
      const dur = Math.round((Date.now() - enterTime) / 1000);
      if (dur >= 3) reportBrowse(dur);
    }
  });
  window.addEventListener('beforeunload', () => {
    const dur = Math.round((Date.now() - enterTime) / 1000);
    if (dur >= 3 && !reported) {
      // beforeunload 里 fetch 不可靠，用 sendBeacon 风格的消息
      const data = collect();
      navigator.sendBeacon?.(''); // no-op 占位，真正上报走 background 定时
    }
  });

  // ===== 消息监听（来自 popup / background）=====
  chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (msg.type === 'COLLECT_PAGE') {
      const data = collect();
      if (data) {
        sendResponse({ ok: true, data });
      } else {
        sendResponse({ ok: false, message: '当前页面无可采集的商品信息' });
      }
      return true; // async
    }
    if (msg.type === 'GET_PAGE_INFO') {
      const data = collect();
      sendResponse({
        ok: true,
        platform: PLATFORM,
        canCollect: !!data,
        preview: data ? { name: data.name, price: data.price, supplier: data.supplier?.name } : null,
      });
      return true;
    }
  });

  // ===== 自动采集（进入商品页后延迟采集，确保 DOM 渲染完成）=====
  setTimeout(() => {
    chrome.storage?.local.get('trademind_config').then(result => {
      const cfg = result.trademind_config || {};
      if (cfg.autoCollect !== false) {
        const data = collect();
        if (data) {
          chrome.runtime?.sendMessage({ type: 'AUTO_COLLECT', data }).catch(() => {});
        }
      }
    });
  }, 2500);
})();
