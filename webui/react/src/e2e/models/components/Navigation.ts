import { NamedComponent } from 'playwright-page-model-base/BaseComponent';

import { NavigationSideBar } from 'e2e/models/components/NavigationSideBar';

/**
 * Represents the Navigation component in src/components/Navigation.tsx
 */
export class Navigation extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="navigation"]';
  // sidebar is desktop view, tabbar is mobile view
  readonly sidebar = new NavigationSideBar({ parent: this });
  // TODO mobile
  // readonly tabbar = new NavigationTabbar({ parent: this });
}
