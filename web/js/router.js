// Hash 路由：轻量、无需服务端配置，刷新/直达均稳定。

import * as dashboard from './views/dashboard.js';
import * as repos from './views/repos.js';
import * as notifications from './views/notifications.js';
import * as settings from './views/settings.js';

const routes = {
  '': dashboard,
  '/': dashboard,
  '/repos': repos,
  '/notifications': notifications,
  '/settings': settings,
};

let contentEl = null;
let current = null;

function highlight(path) {
  document.querySelectorAll('.nav__item').forEach((a) => {
    const to = a.getAttribute('data-route');
    a.classList.toggle('is-active', to === path);
  });
}

function route() {
  const hash = location.hash.replace(/^#/, '');
  let path = hash.split('?')[0] || '/';
  if (path === '') path = '/';
  const view = routes[path] || routes['/'];

  // 先卸载上一个视图（清除其事件监听 / 定时器），避免重复访问时叠加。
  if (current && typeof current.cleanup === 'function') current.cleanup();
  contentEl.innerHTML = '';
  view.render(contentEl);
  current = view;
  highlight(path);

  const main = document.querySelector('.content');
  if (main) main.scrollTop = 0;
}

export function initRouter(el) {
  contentEl = el;
  window.addEventListener('hashchange', route);
  route();
}

export function navigate(path) {
  if (location.hash.replace(/^#/, '') !== path) location.hash = path;
  else route();
}
