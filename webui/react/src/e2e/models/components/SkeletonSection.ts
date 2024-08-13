import { BaseComponent, NamedComponent } from 'playwright-page-model-base/BaseComponent';

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
