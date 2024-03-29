import { BaseComponent, NamedComponent, NamedComponentArgs } from 'e2e/models/BaseComponent';

/**
 * Returns a representation of the SkeletonSection component.
 * This constructor represents the contents in src/components/SkeletonSection.tsx.
 * @param {object} obj
 * @param {implementsGetLocator} obj.parent - The parent used to locate this SkeletonSection
 * @param {string} [obj.selector] - Used instead of `defaultSelector`
 */
export class SkeletonSection extends NamedComponent {
  static defaultSelector = 'div[data-test-component="skeletonSection"]';
  constructor({ selector, parent }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || SkeletonSection.defaultSelector });
  }
  readonly header: BaseComponent = new BaseComponent({
    parent: this,
    selector: '[data-testid="skeletonHeader"]',
  });
}
