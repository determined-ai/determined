import { BaseComponent } from 'playwright-page-model-base/BaseComponent';
import { BaseHeadRow, BaseRow, BaseTable } from 'playwright-page-model-base/BaseTable';

import { Pagination } from 'e2e/models/common/ant/Pagination';
export type { TableArgs } from 'playwright-page-model-base/BaseTable';

/**
 * Represents the Table component from antd/es/table/Table.d.ts.
 */
export class Table<RowType extends Row, HeadRowType extends HeadRow> extends BaseTable<
  RowType,
  HeadRowType
> {
  readonly defaultSelector = '.ant-table';
  readonly noData = new BaseComponent({
    parent: this._body,
    selector: '.ant-empty.ant-empty-normal',
  });
  readonly pagination = new Pagination({
    parent: this._parent,
  });
}

/**
 * Represents a row from the Table component
 */
export class Row extends BaseRow {
  readonly defaultSelector = 'tr.ant-table-row';
  readonly keyAttribute = 'data-row-key';
  readonly select = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column',
  });
}

/**
 * Represents the head row from the Table component
 */
export class HeadRow extends BaseHeadRow {
  readonly defaultSelector = 'tr';
  readonly selectAll = new BaseComponent({
    parent: this,
    selector: '.ant-table-selection-column .ant-checkbox-input',
  });
}
