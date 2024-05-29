import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Tasks Page component.
 * This constructor represents the contents in src/components/Page.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class TasksComponent extends NamedComponent {
  override defaultSelector: string = '[id$=tasks]';
}
