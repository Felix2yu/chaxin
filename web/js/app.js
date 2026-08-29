// 应用入口：构建外壳（侧边栏 + 顶栏），初始化主题 / Toast / 路由。

import { icon } from './ui.js';
import { initTheme } from './theme.js';
import { initToast } from './ui.js';
import { initRouter } from './router.js';

const NAV = [
  { route: '/', label: '总览', icon: 'dashboard' },
  { route: '/repos', label: '仓库管理', icon: 'repos' },
  { route: '/notifications', label: '通知中心', icon: 'notifications' },
  { route: '/settings', label: '系统设置', icon: 'settings' },
];

function shell() {
  const navItems = NAV.map(
    (n) => `
    <a class="nav__item" data-route="${n.route}" href="#${n.route}">
      <span class="nav__icon">${icon(n.icon)}</span>
      <span>${n.label}</span>
      <span class="nav__dot"></span>
    </a>`
  ).join('');

  return `
  <div class="layout">
    <aside class="sidebar" id="sidebar">
      <div class="sidebar__brand">
        <div class="sidebar__logo"><img src="/icons/icon.svg" alt="察新" /></div>
        <div>
          <div class="sidebar__title">察新</div>
          <div class="sidebar__subtitle">版本更新通知</div>
        </div>
      </div>
      <nav class="nav">
        <div class="nav__label">导航菜单</div>
        ${navItems}
      </nav>
    </aside>
    <div class="sidebar-backdrop" id="sidebar-backdrop"></div>
    <div class="main">
      <header class="topbar">
        <button class="topbar__menu" id="menu-toggle" aria-label="菜单">${icon(
          'menu'
        )}</button>
        <div class="topbar__spacer"></div>
        <a class="topbar__icon" href="https://github.com" target="_blank" rel="noopener" title="GitHub">${icon(
          'github'
        )}</a>
      </header>
      <main class="content" id="content"></main>
    </div>
  </div>`;
}

function bindShell() {
  const sidebar = document.getElementById('sidebar');
  const backdrop = document.getElementById('sidebar-backdrop');
  const toggle = document.getElementById('menu-toggle');
  const close = () => {
    sidebar.classList.remove('is-open');
    backdrop.classList.remove('is-open');
  };
  toggle.addEventListener('click', () => {
    sidebar.classList.toggle('is-open');
    backdrop.classList.toggle('is-open');
  });
  backdrop.addEventListener('click', close);
  // 导航后在小屏自动收起抽屉
  document.querySelectorAll('.nav__item').forEach((a) =>
    a.addEventListener('click', close)
  );
}

function start() {
  document.getElementById('app').innerHTML = shell();
  bindShell();
  initTheme();
  initToast();
  initRouter(document.getElementById('content'));
}

start();
