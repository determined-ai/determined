import { Popover } from 'antd';
import * as io from 'io-ts';

import Button from 'components/kit/Button';
import { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import Select from 'components/kit/Select';
import { V1ColumnType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';
import { Loadable } from 'utils/loadable';

import css from './MultiSortMenu.module.scss';

// in the list of columns from the api but not supported by the sort functionality
const BANNED_SORT_COLUMNS = new Set(['tags', 'searcherMetric']);

const directionType = io.keyof({ asc: null, desc: null });
export type DirectionType = io.TypeOf<typeof directionType>;

export const validSort = io.type({
  column: io.string,
  direction: directionType,
});
export type ValidSort = io.TypeOf<typeof validSort>;

const sort = io.partial(validSort.props);
export type Sort = io.TypeOf<typeof sort>;

export const EMPTY_SORT: Sort = { column: undefined, direction: undefined };

interface MultiSortProps {
  sorts: Sort[];
  columns: Loadable<ProjectColumn[]>;
  onChange: (sorts: Sort[]) => void;
}
interface MultiSortRowProps {
  sort: Sort;
  columns: Loadable<ProjectColumn[]>;
  onChange: (sort: Sort) => void;
  onRemove: () => void;
}
interface DirectionOptionsProps {
  onChange: (direction: DirectionType) => void;
  type: V1ColumnType;
  value?: DirectionType;
}
interface ColumnOptionsProps {
  onChange: (column: string) => void;
  columns: Loadable<ProjectColumn[]>;
  value?: string;
}

export const optionsByColumnType = {
  [V1ColumnType.NUMBER]: [
    { label: '0 → 9', value: 'asc' },
    { label: '9 → 0', value: 'desc' },
  ],
  [V1ColumnType.TEXT]: [
    { label: 'A → Z', value: 'asc' },
    { label: 'Z → A', value: 'desc' },
  ],
  [V1ColumnType.DATE]: [
    { label: 'Newest → Oldest', value: 'desc' },
    { label: 'Oldest → Newest', value: 'asc' },
  ],
  [V1ColumnType.UNSPECIFIED]: [
    { label: 'Ascending', value: 'asc' },
    { label: 'Descending', value: 'desc' },
  ],
};

const SortArrow = ({ direction = 'asc' }: { direction: DirectionType }) => (
  <svg
    className={css.sortIcon + ' ' + (css[`sortIcon--${direction}`] || '')}
    fill="none"
    height="1em"
    viewBox="0 0 240 240"
    width="1em"
    xmlns="http://www.w3.org/2000/svg">
    <g stroke="currentcolor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="10">
      <path d="M180 80L120 20L60 80" />
      <path d="M120 25L120 220" />
    </g>
  </svg>
);

const SortButtonIcon = () => (
  <svg
    className="anticon"
    fill="none"
    height="1em"
    viewBox="0 0 240 240"
    width="1em"
    xmlns="http://www.w3.org/2000/svg">
    <g stroke="currentcolor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="10">
      <path d="M108.5 85.5001L60 37.0001L11.5 85.5001" />
      <path d="M60 42L60 202" />
      <path d="M133 153.5L181.5 202L230 153.5" />
      <path d="M181.5 197L181.5 37.0001" />
    </g>
  </svg>
);

export const sortMenuItemsForColumn = (
  column: ProjectColumn,
  sorts: Sort[],
  onSortChange: (sorts: Sort[]) => void,
): MenuItem[] => {
  if (BANNED_SORT_COLUMNS.has(column.column)) {
    return [];
  }
  return optionsByColumnType[column.type].map((option) => {
    const curSort = sorts.find((s) => s.column === column.column);
    const isSortMatch = curSort && curSort.direction === option.value;
    return {
      icon: <SortArrow direction={option.value as DirectionType} />,
      key: option.value,
      label: `Sort ${option.label}`,
      onClick: () => {
        let newSort: Sort[];
        if (isSortMatch) {
          return;
        } else if (curSort) {
          newSort = sorts.map((s) =>
            s.column !== column.column
              ? s
              : {
                  ...s,
                  direction: option.value as DirectionType,
                },
          );
        } else {
          newSort = [{ column: column.column, direction: option.value as DirectionType }];
        }
        onSortChange(newSort);
      },
    };
  });
};

const DirectionOptions: React.FC<DirectionOptionsProps> = ({ onChange, type, value }) => (
  <Select
    options={optionsByColumnType[type]}
    placeholder="Select direction"
    value={value}
    width="100%"
    onChange={(val) => onChange(val as DirectionType)}
  />
);

const ColumnOptions: React.FC<ColumnOptionsProps> = ({ onChange, columns, value }) => (
  <Select
    autoFocus
    loading={Loadable.isLoading(columns)}
    options={Loadable.getOrElse([], columns)
      .filter((c) => !BANNED_SORT_COLUMNS.has(c.column))
      .map((c) => ({
        label: c.displayName || c.column,
        value: c.column,
      }))}
    placeholder="Select column"
    value={value}
    width="100%"
    onChange={(val) => onChange(val as string)}
  />
);

const MultiSortRow: React.FC<MultiSortRowProps> = ({ sort, columns, onChange, onRemove }) => {
  const valueType =
    Loadable.getOrElse([], columns).find((c) => c.column === sort.column)?.type ||
    V1ColumnType.UNSPECIFIED;
  return (
    <div className={css.sortRow}>
      <div className={css.select}>
        <ColumnOptions
          columns={columns}
          value={sort.column}
          onChange={(column) => onChange({ ...sort, column })}
        />
      </div>
      <div className={css.select}>
        <DirectionOptions
          type={valueType}
          value={sort.direction}
          onChange={(direction) => onChange({ ...sort, direction })}
        />
      </div>
      <div>
        <Button
          icon={<Icon name="close" title="Remove sort" />}
          size="small"
          type="text"
          onClick={onRemove}
        />
      </div>
    </div>
  );
};

const MultiSort: React.FC<MultiSortProps> = ({ sorts, columns, onChange }) => {
  const makeOnRowChange = (idx: number) => (sort: Sort) => {
    const newSorts = [...sorts];
    newSorts[idx] = {
      ...sort,
      direction: sort.direction || 'asc',
    };
    onChange(newSorts);
  };
  const makeOnRowRemove = (idx: number) => () => {
    const newSorts = sorts.filter((_, cidx) => cidx !== idx);
    onChange(newSorts.length > 0 ? newSorts : [EMPTY_SORT]);
  };
  const addRow = () => onChange([...sorts, EMPTY_SORT]);
  const clearAll = () => {
    // close the popover -- happens before the onchange so the onclose handler fires first
    window.document.body.dispatchEvent(new Event('mousedown', { bubbles: true }));
    onChange([EMPTY_SORT]);
  };

  return (
    <div className={css.base}>
      <div>Sort by</div>
      <div className={css.rows}>
        {sorts.map((sort, idx) => {
          const seenColumns = sorts.slice(0, idx).map((s) => s.column);
          const columnOptions = Loadable.map(columns, (cols) =>
            cols.filter((c) => !seenColumns.includes(c.column)),
          );
          return (
            <MultiSortRow
              columns={columnOptions}
              key={sort.column || idx}
              sort={sort}
              onChange={makeOnRowChange(idx)}
              onRemove={makeOnRowRemove(idx)}
            />
          );
        })}
      </div>
      <div className={css.actions}>
        <Button
          icon={<Icon decorative name="add-small" size="tiny" />}
          type="text"
          onClick={addRow}>
          Add sort
        </Button>
        <Button type="text" onClick={clearAll}>
          Reset
        </Button>
      </div>
    </div>
  );
};

const MultiSortMenu: React.FC<MultiSortProps> = ({ sorts, columns, onChange }) => {
  const validSorts = sorts.filter(validSort.is);
  const onSortPopoverOpenChange = (open: boolean) => {
    if (!open) {
      onChange(validSorts.length > 0 ? validSorts : [EMPTY_SORT]);
    }
  };

  return (
    <Popover
      content={<MultiSort columns={columns} sorts={sorts} onChange={onChange} />}
      placement="bottomLeft"
      showArrow={false}
      trigger="click"
      onOpenChange={onSortPopoverOpenChange}>
      <Button icon={<SortButtonIcon />}>
        Sort {validSorts.length ? `(${validSorts.length})` : ''}
      </Button>
    </Popover>
  );
};

export default MultiSortMenu;
