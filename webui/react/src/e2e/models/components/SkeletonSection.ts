import { BaseComponent, NamedComponent } from 'e2e/models/common/base/BaseComponent';

/**
 * Represents the SkeletonSection component in src/components/SkeletonSection.tsx
 */
export class SkeletonSection extends NamedComponent {
  readonly defaultSelector = 'div[data-test-component="skeletonSection"]';
  readonly header = new BaseComponent({
    parent: this,
    selector: '[data-testid="skeletonHeader"]',
  });
}
