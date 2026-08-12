// 用户认证接口测试端前端交互逻辑。

const API_BASE = '';
const historyList = [];

// fetchAPI 封装后端API调用。
async function fetchAPI(path, method, body) {
    const opts = { method: method || 'GET', headers: { 'Content-Type': 'application/json' } };
    if (body) {
        opts.body = JSON.stringify(body);
    }
    try {
        const resp = await fetch(API_BASE + path, opts);
        const data = await resp.json();
        return data;
    } catch (err) {
        return { success: false, error_msg: '请求失败: ' + err.message };
    }
}

// formatJSON 格式化JSON展示。
function formatJSON(obj) {
    try {
        return JSON.stringify(obj, null, 2);
    } catch {
        return String(obj);
    }
}

// addHistory 添加请求历史记录。
function addHistory(type, req, resp, durationMs) {
    const entry = {
        time: new Date().toLocaleTimeString(),
        type: type,
        request: req,
        response: resp,
        durationMs: durationMs,
    };
    historyList.unshift(entry);
    if (historyList.length > 50) {
        historyList.pop();
    }
    renderHistory();
}

// renderHistory 渲染请求历史。
function renderHistory() {
    const container = document.getElementById('history-list');
    if (historyList.length === 0) {
        container.innerHTML = '<div class="history-item">暂无历史记录</div>';
        return;
    }
    container.innerHTML = historyList.map(function(e) {
        const cls = e.response && e.response.success ? 'success' : 'failure';
        const reqStr = formatJSON(e.request).replace(/"password"\s*:\s*"[^"]*"/g, '"password":"***"');
        return '<div class="history-item ' + cls + '">' +
            '<span class="time">[' + e.time + '] ' + e.type + '</span>' +
            '<div class="detail">请求: ' + reqStr + '</div>' +
            '<div class="detail">响应: ' + formatJSON(e.response) + '</div>' +
            '</div>';
    }).join('');
}

// showResult 展示结果到指定区域。
function showResult(elemId, data) {
    const elem = document.getElementById(elemId);
    if (data.success) {
        elem.innerHTML = '<span class="status-success">成功</span>\n' + formatJSON(data.data || data);
    } else {
        elem.innerHTML = '<span class="status-failure">失败</span>\n' + (data.error_msg || '未知错误');
    }
}

// 环境初始化按钮绑定。
document.getElementById('btn-env-init').addEventListener('click', async function() {
    const data = await fetchAPI('/api/env/init', 'POST');
    showResult('env-result', data);
});

document.getElementById('btn-env-status').addEventListener('click', async function() {
    const data = await fetchAPI('/api/env/status', 'GET');
    showResult('env-result', data);
});

document.getElementById('btn-env-cleanup').addEventListener('click', async function() {
    if (!confirm('确认清理测试数据？将清空所有表数据但保留表结构。')) return;
    const data = await fetchAPI('/api/env/cleanup', 'POST');
    showResult('env-result', data);
});

document.getElementById('btn-env-destroy').addEventListener('click', async function() {
    if (!confirm('确认销毁测试库？此操作不可恢复！')) return;
    const data = await fetchAPI('/api/env/destroy', 'POST');
    showResult('env-result', data);
});

// 服务管理按钮绑定。
document.getElementById('btn-sut-start').addEventListener('click', async function() {
    const data = await fetchAPI('/api/sut/start', 'POST');
    showResult('sut-status', data);
});

document.getElementById('btn-sut-stop').addEventListener('click', async function() {
    const data = await fetchAPI('/api/sut/stop', 'POST');
    showResult('sut-status', data);
});

document.getElementById('btn-sut-status').addEventListener('click', async function() {
    const data = await fetchAPI('/api/sut/status', 'GET');
    const elem = document.getElementById('sut-status');
    if (data.success) {
        const s = data.data;
        const statusMap = {1: '未启动', 2: '启动中', 3: '运行中', 4: '已停止', 5: '异常退出'};
        elem.innerHTML = '<span class="status-success">状态: ' + (statusMap[s.Status] || '未知') +
            '</span> | PID: ' + s.PID + ' | 退出码: ' + s.ExitCode;
        const logs = document.getElementById('sut-logs');
        logs.innerHTML = (s.Logs || []).join('\n');
    } else {
        elem.innerHTML = '<span class="status-failure">查询失败: ' + (data.error_msg || '') + '</span>';
    }
});

// 认证测试发送函数。
async function sendAuth(type, fields) {
    const body = { type: type };
    for (const k in fields) {
        body[k] = fields[k];
    }
    const data = await fetchAPI('/api/auth/send', 'POST', body);
    const elem = document.getElementById('auth-result');
    if (data.success) {
        const r = data.data;
        const resp = r.Response || r.response || r;
        const dur = r.DurationMs || r.duration_ms || 0;
        if (resp && resp.success) {
            elem.innerHTML = '<span class="status-success">成功</span> | 耗时: ' + dur + 'ms\n' + formatJSON(resp);
        } else {
            elem.innerHTML = '<span class="status-failure">失败</span> | 耗时: ' + dur + 'ms\n' + formatJSON(resp);
        }
        addHistory(type, body, resp, dur);
        return resp;
    } else {
        elem.innerHTML = '<span class="status-failure">错误: ' + (data.error_msg || '') + '</span>';
        addHistory(type, body, data, 0);
        return null;
    }
}

// 注册按钮。
document.getElementById('btn-register').addEventListener('click', async function() {
    const username = document.getElementById('reg-username').value.trim();
    const password = document.getElementById('reg-password').value.trim();
    if (!username || !password) {
        alert('请输入用户名和密码');
        return;
    }
    const resp = await sendAuth('register', { username: username, password: password });
    if (resp && resp.success && resp.player_id) {
        document.getElementById('login-username').value = username;
        document.getElementById('login-password').value = password;
    }
});

// 登录按钮。
document.getElementById('btn-login').addEventListener('click', async function() {
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value.trim();
    const deviceId = document.getElementById('login-device-id').value.trim();
    if (!username || !password) {
        alert('请输入用户名和密码');
        return;
    }
    const resp = await sendAuth('login', { username: username, password: password, device_id: deviceId });
    if (resp && resp.success && resp.token) {
        document.getElementById('hb-token').value = resp.token;
        document.getElementById('logout-token').value = resp.token;
        if (resp.player_id) {
            document.getElementById('hb-player-id').value = resp.player_id;
            document.getElementById('logout-player-id').value = resp.player_id;
        }
    }
    }
});

// 心跳按钮。
document.getElementById('btn-heartbeat').addEventListener('click', async function() {
    const token = document.getElementById('hb-token').value.trim();
    const playerId = parseInt(document.getElementById('hb-player-id').value) || 0;
    if (!token) {
        alert('请输入令牌');
        return;
    }
    await sendAuth('heartbeat', { token: token, player_id: playerId });
});

// 登出按钮。
document.getElementById('btn-logout').addEventListener('click', async function() {
    const token = document.getElementById('logout-token').value.trim();
    const playerId = parseInt(document.getElementById('logout-player-id').value) || 0;
    if (!token) {
        alert('请输入令牌');
        return;
    }
    await sendAuth('logout', { token: token, player_id: playerId });
});

// 端到端测试按钮。
document.getElementById('btn-e2e-run').addEventListener('click', async function() {
    const body = {
        username: document.getElementById('e2e-username').value.trim(),
        password: document.getElementById('e2e-password').value.trim(),
        device_id: document.getElementById('e2e-device-id').value.trim(),
    };
    const elem = document.getElementById('e2e-report');
    elem.innerHTML = '<span class="status-timeout">测试执行中...</span>';
    const data = await fetchAPI('/api/e2e/run', 'POST', body);
    if (data.success) {
        const report = data.data;
        const statusCls = report.OverallPassed ? 'status-success' : 'status-failure';
        let html = '<span class="' + statusCls + '">通过率: ' + report.PassRate + '% | ' +
            (report.OverallPassed ? '全部通过' : '存在失败') + '</span>\n';
        html += '<table><tr><th>步骤</th><th>状态</th><th>耗时(ms)</th><th>断言</th></tr>';
        const statusMap = {1: '通过', 2: '失败', 3: '跳过'};
        for (const step of report.Steps) {
            const cls = step.Status === 1 ? 'status-success' : (step.Status === 2 ? 'status-failure' : 'status-timeout');
            html += '<tr><td>' + step.StepName + '</td><td class="' + cls + '">' +
                (statusMap[step.Status] || '未知') + '</td><td>' + step.DurationMs + '</td><td>' +
                (step.Assertion || '') + '</td></tr>';
        }
        html += '</table>';
        elem.innerHTML = html;
    } else {
        elem.innerHTML = '<span class="status-failure">测试失败: ' + (data.error_msg || '') + '</span>';
    }
});

// 失败场景测试按钮。
document.getElementById('btn-e2e-failure').addEventListener('click', async function() {
    const elem = document.getElementById('e2e-report');
    elem.innerHTML = '<span class="status-timeout">失败场景执行中...</span>';
    const data = await fetchAPI('/api/e2e/failure', 'POST');
    if (data.success) {
        const results = data.data || [];
        let html = '<table><tr><th>用例</th><th>状态</th><th>断言</th></tr>';
        const statusMap = {1: '通过', 2: '失败', 3: '跳过'};
        for (const r of results) {
            const cls = r.Status === 1 ? 'status-success' : 'status-failure';
            html += '<tr><td>' + r.StepName + '</td><td class="' + cls + '">' +
                (statusMap[r.Status] || '未知') + '</td><td>' + (r.Assertion || '') + '</td></tr>';
        }
        html += '</table>';
        elem.innerHTML = html;
    } else {
        elem.innerHTML = '<span class="status-failure">执行失败: ' + (data.error_msg || '') + '</span>';
    }
});

// 清空历史按钮。
document.getElementById('btn-clear-history').addEventListener('click', function() {
    historyList.length = 0;
    renderHistory();
});

// 初始化。
renderHistory();
