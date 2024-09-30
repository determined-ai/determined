import { BaseComponent } from 'playwright-page-model-base/BaseComponent';

import { Modal } from 'e2e/models/common/hew/Modal';

/**
 * Represents the ModelCreateModal component in src/components/ModelCreateModal.tsx
 */
export class ModelCreateModal extends Modal {
  readonly name = new BaseComponent({
    parent: this,
    selector: '[id="modelName"]',
  });
  readonly description = new BaseComponent({
    parent: this,
    selector: '[id="modelDescription"]',
  });
  readonly addMoreDetails = new BaseComponent({
    parent: this,
    selector: '[class^="Link_base"]',
  });
  readonly addMetadatButton = new BaseComponent({
    parent: this,
    selector: '[test-id="add-metadata"]',
  });
  readonly addTagButton = new BaseComponent({
    parent: this,
    selector: '[test-id="add-tag"]',
  });
  readonly metadataKey = new BaseComponent({
    parent: this,
    selector: '[id="metadata_0_key"]',
  });
  readonly metadataValue = new BaseComponent({
    parent: this,
    selector: '[id="metadata_0_value"]',
  });
  readonly tag = new BaseComponent({
    parent: this,
    selector: '[id="tags_0"]',
  });
}
