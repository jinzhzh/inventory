(() => {
  'use strict';

  const $ = (s) => document.querySelector(s);
  const $$ = (s) => document.querySelectorAll(s);
  const API = '';

  async function get(path) {
    const r = await fetch(API + path);
    if (!r.ok) throw new Error((await r.json()).error || r.statusText);
    return r.json();
  }

  async function post(path, body) {
    const r = await fetch(API + path, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!r.ok) throw new Error((await r.json()).error || r.statusText);
    return r.json();
  }

  async function del(path) {
    const r = await fetch(API + path, { method: 'DELETE' });
    if (!r.ok) throw new Error((await r.json()).error || r.statusText);
    return r.json();
  }

  // ---- Navigation ----
  $$('#tabs .tab').forEach((btn) => {
    btn.addEventListener('click', () => {
      $$('.tab').forEach((b) => b.classList.remove('active'));
      $$('.page').forEach((p) => p.classList.remove('active'));
      btn.classList.add('active');
      $(`#page-${btn.dataset.page}`).classList.add('active');
      loadPage(btn.dataset.page);
    });
  });

  async function loadPage(page) {
    try {
      switch (page) {
        case 'dashboard': await loadDashboard(); break;
        case 'skus': await loadSKUs(); break;
        case 'inbound': await loadSKUDropdowns(); break;
        case 'outbound': await loadSKUDropdowns(); break;
        case 'inventory': await loadInventory(); break;
        case 'ledger': await loadLedger(); break;
      }
    } catch (e) {
      console.error(e);
    }
  }

  // ---- Dashboard ----
  async function loadDashboard() {
    const stats = await get('/api/stats');
    $('#statsGrid').innerHTML = [
      { v: stats.total_sku, l: 'SKU总数' },
      { v: stats.total_in, l: '总入库' },
      { v: stats.total_out, l: '总出库' },
      { v: stats.alert_count, l: '低库存预警' },
      { v: '¥' + (stats.total_value || 0).toFixed(2), l: '库存总值' },
    ].map((s) => `<div class="stat-card"><div class="val">${s.v}</div><div class="label">${s.l}</div></div>`).join('');

    const alerts = await get('/api/alerts');
    if (alerts.length) {
      $('#alertBadge').style.display = '';
      $('#alertCount').textContent = alerts.length;
      $('#alertsList').innerHTML = alerts.map((a) =>
        `<div class="alert-item"><span class="name">${esc(a.name)}（${a.sku_code}）</span><span class="qty">库存 ${a.current_qty} / 预警 ${a.alert_qty} ${a.unit}</span></div>`
      ).join('');
    } else {
      $('#alertBadge').style.display = 'none';
      $('#alertsList').innerHTML = '<p class="empty">暂无预警，库存充足 ✅</p>';
    }
  }

  // ---- SKU ----
  async function loadSKUs() {
    const skus = await get('/api/skus');
    if (!skus.length) {
      $('#skuList').innerHTML = '<p class="empty">暂无SKU，点击"新增SKU"添加。</p>';
      return;
    }
    $('#skuList').innerHTML = `<table><thead><tr><th>名称</th><th>编码</th><th>分类</th><th>库存</th><th>成本价</th><th>售价</th><th>操作</th></tr></thead><tbody>` +
      skus.map((s) => `<tr>
        <td>${esc(s.name)}</td><td>${esc(s.sku_code)}</td><td>${esc(s.category)}</td>
        <td class="${s.current_qty <= s.alert_qty ? 'low' : 'ok'}">${s.current_qty} ${esc(s.unit)}</td>
        <td>¥${s.cost_price.toFixed(2)}</td><td>¥${s.sale_price.toFixed(2)}</td>
        <td><button class="btn btn-sm" onclick="editSKU('${s.id}')">编辑</button> <button class="btn btn-sm btn-danger" onclick="deleteSKU('${s.id}')">删除</button></td>
      </tr>`).join('') + `</tbody></table>`;
  }

  window.editSKU = async function (id) {
    const s = await get('/api/skus/' + id);
    $('#skuId').value = s.id;
    $('#skuName').value = s.name;
    $('#skuCode').value = s.sku_code;
    $('#skuCat').value = s.category || '';
    $('#skuUnit').value = s.unit;
    $('#skuAlert').value = s.alert_qty;
    $('#skuCost').value = s.cost_price;
    $('#skuSale').value = s.sale_price;
    $('#modalTitle').textContent = '编辑SKU';
    $('#skuModal').classList.add('open');
  };

  window.deleteSKU = async function (id) {
    if (!confirm('确定删除此SKU？')) return;
    try {
      await del('/api/skus/' + id);
      loadSKUs();
    } catch (e) { alert(e.message); }
  };

  $('#btnAddSKU').addEventListener('click', () => {
    $('#skuForm').reset();
    $('#skuId').value = '';
    $('#modalTitle').textContent = '新增SKU';
    $('#skuModal').classList.add('open');
  });
  $('#btnCancelSKU').addEventListener('click', () => $('#skuModal').classList.remove('open'));

  $('#skuForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const body = {
      name: $('#skuName').value.trim(),
      sku_code: $('#skuCode').value.trim(),
      category: $('#skuCat').value.trim(),
      unit: $('#skuUnit').value.trim() || '个',
      alert_qty: parseInt($('#skuAlert').value) || 10,
      cost_price: parseFloat($('#skuCost').value) || 0,
      sale_price: parseFloat($('#skuSale').value) || 0,
    };
    if (!body.name || !body.sku_code) { $('#skuError').textContent = '名称和编码必填'; return; }
    try {
      const id = $('#skuId').value;
      if (id) {
        await fetch(API + '/api/skus/' + id, {
          method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
        });
      } else {
        await post('/api/skus', body);
      }
      $('#skuModal').classList.remove('open');
      loadSKUs();
    } catch (err) { $('#skuError').textContent = err.message; }
  });

  // ---- Dropdowns ----
  async function loadSKUDropdowns() {
    const skus = await get('/api/skus');
    const opts = '<option value="">-- 选择SKU --</option>' + skus.map((s) =>
      `<option value="${s.id}">${esc(s.name)}（${esc(s.sku_code)}）库存:${s.current_qty}`
    ).join('');
    $('#inSKU').innerHTML = opts;
    $('#outSKU').innerHTML = opts;
  }

  // ---- Inbound ----
  $('#inboundForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    $('#inError').textContent = '';
    try {
      await post('/api/inbound', {
        sku: $('#inSKU').value,
        quantity: parseInt($('#inQty').value),
        price: parseFloat($('#inPrice').value) || 0,
        remark: $('#inRemark').value.trim(),
        operator: $('#inOp').value.trim(),
      });
      alert('✅ 入库成功');
      $('#inboundForm').reset();
      loadSKUDropdowns();
    } catch (err) { $('#inError').textContent = err.message; }
  });

  // ---- Outbound ----
  $('#outboundForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    $('#outError').textContent = '';
    try {
      await post('/api/outbound', {
        sku: $('#outSKU').value,
        quantity: parseInt($('#outQty').value),
        price: parseFloat($('#outPrice').value) || 0,
        remark: $('#outRemark').value.trim(),
        operator: $('#outOp').value.trim(),
      });
      alert('✅ 出库成功');
      $('#outboundForm').reset();
      loadSKUDropdowns();
    } catch (err) { $('#outError').textContent = err.message; }
  });

  // ---- Inventory ----
  async function loadInventory() {
    const inv = await get('/api/inventory');
    if (!inv.length) { $('#inventoryTable').innerHTML = '<p class="empty">暂无库存数据</p>'; return; }
    $('#inventoryTable').innerHTML = `<table><thead><tr><th>名称</th><th>编码</th><th>分类</th><th>当前库存</th><th>预警线</th><th>状态</th></tr></thead><tbody>` +
      inv.map((i) => `<tr>
        <td>${esc(i.name)}</td><td>${esc(i.sku_code)}</td><td>${esc(i.category)}</td>
        <td class="${i.current_qty <= i.alert_qty ? 'low' : 'ok'}">${i.current_qty} ${esc(i.unit)}</td>
        <td>${i.alert_qty} ${esc(i.unit)}</td>
        <td>${i.current_qty <= i.alert_qty ? '⚠️ 低库存' : '✅ 正常'}</td>
      </tr>`).join('') + `</tbody></table>`;
  }

  // ---- Ledger ----
  async function loadLedger() {
    const txs = await get('/api/ledger?limit=100');
    if (!txs.length) { $('#ledgerTable').innerHTML = '<p class="empty">暂无出入库记录</p>'; return; }
    $('#ledgerTable').innerHTML = `<table><thead><tr><th>时间</th><th>商品</th><th>类型</th><th>数量</th><th>单价</th><th>金额</th><th>备注</th><th>操作人</th></tr></thead><tbody>` +
      txs.map((t) => `<tr>
        <td>${new Date(t.created_at).toLocaleString('zh-CN')}</td>
        <td>${esc(t.sku_name)}</td>
        <td><span class="badge ${t.type === 'inbound' ? 'badge-in' : 'badge-out'}">${t.type === 'inbound' ? '入库' : '出库'}</span></td>
        <td>${t.quantity}</td>
        <td>¥${t.price.toFixed(2)}</td>
        <td>¥${t.total.toFixed(2)}</td>
        <td>${esc(t.remark)}</td>
        <td>${esc(t.operator)}</td>
      </tr>`).join('') + `</tbody></table>`;
  }

  function esc(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  // Init
  loadDashboard();
})();