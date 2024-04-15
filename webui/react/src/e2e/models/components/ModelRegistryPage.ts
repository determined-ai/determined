import { NamedComponent } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the Model Registry Page component.
 * This constructor represents the contents in src/components/ModelRegistry.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this Page
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class ModelRegistryComponent extends NamedComponent {
  defaultSelector: string = '[data-testid=modelRegistry]';
}
