// Copyright (c) 2026 AllIsA1. SPDX-License-Identifier: MIT
(function () {
  'use strict';

  var TEXT = {
    en: {
      title: 'OpenFlux core',
      intro:
        'Keeps the OpenFlux exit program of the panel at the version this plugin was reviewed with. The panel downloads it itself and checks its hash; exits that use it restart on the panel’s next pass. A core you uploaded yourself on Settings → Cores is left alone until the plugin brings a newer one.',
      install: 'Install now',
      refresh: 'Refresh',
      kept: 'Version kept by the plugin',
      installed: 'Installed by the plugin',
      none: 'nothing yet',
      state: 'Core in the panel',
      stateCurrent: 'the plugin’s version',
      statePresent: 'another core (not installed by the plugin)',
      stateMissing: 'none',
      running: 'Exits running',
      yes: 'yes',
      no: 'no',
      arch: 'Architecture',
      checked: 'Last check',
      error: 'Last problem',
      busy: 'Working…',
      done: 'Done.',
      failed: function (e) { return 'Failed: ' + e; },
    },
    ru: {
      title: 'Ядро OpenFlux',
      intro:
        'Держит программу выхода OpenFlux в панели на версии, с которой проверен этот плагин. Панель скачивает файл сама и сверяет хеш; выходы, которые его используют, перезапускаются при ближайшем проходе панели. Ядро, загруженное вами вручную в Settings → Cores, остаётся как есть, пока плагин не принесёт более новое.',
      install: 'Установить сейчас',
      refresh: 'Обновить',
      kept: 'Версия плагина',
      installed: 'Установлено плагином',
      none: 'пока ничего',
      state: 'Ядро в панели',
      stateCurrent: 'версия плагина',
      statePresent: 'другое ядро (поставлено не плагином)',
      stateMissing: 'нет',
      running: 'Выходы запущены',
      yes: 'да',
      no: 'нет',
      arch: 'Архитектура',
      checked: 'Последняя проверка',
      error: 'Последняя проблема',
      busy: 'Работаю…',
      done: 'Готово.',
      failed: function (e) { return 'Не удалось: ' + e; },
    },
  };
  var lang = 'en';
  var last = null;

  function T() { return TEXT[lang] || TEXT.en; }
  function $(id) { return document.getElementById(id); }

  function message(text, kind) {
    var m = $('message');
    m.textContent = text || '';
    m.className = 'message' + (kind ? ' ' + kind : '');
  }

  function applyText() {
    document.querySelectorAll('[data-i18n]').forEach(function (n) { n.textContent = T()[n.getAttribute('data-i18n')]; });
    if (last) render(last);
  }

  function row(label, value) {
    var tr = document.createElement('tr');
    var th = document.createElement('th');
    th.textContent = label;
    var td = document.createElement('td');
    td.textContent = value;
    tr.appendChild(th);
    tr.appendChild(td);
    return tr;
  }

  function render(s) {
    last = s;
    var t = T();
    var body = $('facts').querySelector('tbody');
    body.textContent = '';
    body.appendChild(row(t.kept, s.want));
    body.appendChild(row(t.installed, s.installed || t.none));
    body.appendChild(row(t.state, s.current ? t.stateCurrent : s.present ? t.statePresent : t.stateMissing));
    body.appendChild(row(t.running, s.running ? t.yes : t.no));
    body.appendChild(row(t.arch, s.arch));
    body.appendChild(row(t.checked, s.checkedAt && s.checkedAt.indexOf('0001') !== 0 ? new Date(s.checkedAt).toLocaleString() : '—'));
    if (s.lastError) body.appendChild(row(t.error, s.lastError));
  }

  function load() {
    return lucx.api('/api/status').then(function (res) {
      if (!res.ok) return message(T().failed(res.error || res.status), 'error');
      render(res.body);
    });
  }

  $('refresh').onclick = function () { message(''); load(); };
  $('install').onclick = function () {
    $('install').disabled = true;
    message(T().busy);
    lucx.api('/api/install', { method: 'POST', body: {} }).then(function (res) {
      $('install').disabled = false;
      if (!res.ok) {
        message(T().failed((res.body && res.body.error) || res.error || res.status), 'error');
        return load();
      }
      render(res.body);
      message(T().done, 'ok');
    });
  };

  function setLang(ctx) {
    lang = ctx && ctx.lang && String(ctx.lang).slice(0, 2) === 'ru' ? 'ru' : 'en';
    lucx.applyTheme();
    applyText();
  }

  lucx.ready().then(function (ctx) { setLang(ctx); load(); });
  lucx.onContext(setLang);
})();
