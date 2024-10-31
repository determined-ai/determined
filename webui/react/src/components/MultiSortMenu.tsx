import Badge from 'hew/Badge';
import Button from 'hew/Button';
import { DirectionType, Sort, validSort } from 'hew/DataGrid/DataGrid';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Select, { Option } from 'hew/Select';
import { Loadable } from 'hew/utils/loadable';
import { groupBy, mapValues } from 'lodash';
import { Fragment, useMemo } from 'react';

import { runColumns } from 'pages/FlatRuns/columns';
import { V1ColumnType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';
import { removeColumnTypePrefix } from 'utils/flatRun';

import css from './MultiSortMenu.module.scss';

export const EMPTY_SORT: Sort = { column: undefined, direction: undefined };

interface MultiSortProps {
  columns: Loadable<ProjectColumn[]>;
  isMobile?: boolean;
  onChange?: (sorts: Sort[]) => void;
  sorts: Sort[];
  bannedSortColumns?: Set<string>;
}
interface MultiSortRowProps {
  sort: Sort;
  columns: Loadable<ProjectColumn[]>;
  onChange?: (sort: Sort) => void;
  onRemove?: () => void;
  bannedSortColumns: Set<string>;
  typeMap: Record<string, V1ColumnType[]>;
}
interface DirectionOptionsProps {
  onChange?: (direction: DirectionType) => void;
  type: V1ColumnType;
  value?: DirectionType;
}
interface ColumnOptionsProps {
  columns: Loadable<ProjectColumn[]>;
  onChange?: (column: string) => void;
  value?: string;
  bannedSortColumns: Set<string>;
  typeMap: Record<string, V1ColumnType[]>;
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
  [V1ColumnType.ARRAY]: [],
  [V1ColumnType.UNSPECIFIED]: [
    { label: 'Ascending', value: 'asc' },
    { label: 'Descending', value: 'desc' },
  ],
};

export const ADD_SORT_TEXT = 'Add sort';
export const SORT_MENU_TITLE = 'Sort by';
export const RESET_SORT_TEXT = 'Reset';
export const REMOVE_SORT_TITLE = 'Remove sort';
export const SORT_MENU_BUTTON = 'sort-menu-button';

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
  onSortChange?: (sorts: Sort[]) => void,
): MenuItem[] => {
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
        onSortChange?.(newSort);
      },
    };
  });
};

const DirectionOptions: React.FC<DirectionOptionsProps> = ({ onChange, type, value }) => {
  const options = optionsByColumnType[type];
  return (
    <Select
      data-test="direction"
      placeholder="Select direction"
      value={value}
      width="100%"
      onChange={(val) => onChange?.(val as DirectionType)}>
      {options.map((o) => (
        <Option data-select-value={o.value} key={o.value}>
          {o.label}
        </Option>
      ))}
    </Select>
  );
};

const ColumnOptions: React.FC<ColumnOptionsProps> = ({
  onChange,
  columns,
  value,
  bannedSortColumns,
  typeMap,
}) => (
  <Select
    autoFocus
    data-test="column"
    dropdownMatchSelectWidth={300}
    loading={Loadable.isNotLoaded(columns)}
    options={Loadable.getOrElse([], columns)
      .filter((c) => !bannedSortColumns.has(c.column))
      .map((c) => {
        const badges = typeMap[c.column].map((type) => {
          const copy =
            (runColumns as readonly string[]).includes(c.column) &&
            type === V1ColumnType.UNSPECIFIED
              ? 'BOOLEAN'
              : removeColumnTypePrefix(type);
          return (
            <Fragment key={type}>
              <Badge text={copy} />{' '}
            </Fragment>
          );
        });
        const title = c.displayName || c.column;
        const label = (
          <>
            {title} {badges}
          </>
        );
        return {
          label,
          title,
          value: c.column,
        };
      })}
    placeholder="Select column"
    value={value}
    width="100%"
    onChange={(val) => onChange?.(val as string)}
  />
);

const MultiSortRow: React.FC<MultiSortRowProps> = ({
  sort,
  columns,
  onChange,
  onRemove,
  bannedSortColumns,
  typeMap,
}) => {
  const valueType =
    Loadable.getOrElse([], columns).find((c) => c.column === sort.column)?.type ||
    V1ColumnType.UNSPECIFIED;
  return (
    <div className={css.sortRow} data-test-component="multiSortRow">
      <div className={css.select}>
        <ColumnOptions
          bannedSortColumns={bannedSortColumns}
          columns={columns}
          typeMap={typeMap}
          value={sort.column}
          onChange={(column) => onChange?.({ ...sort, column })}
        />
      </div>
      <div className={css.select}>
        <DirectionOptions
          type={valueType}
          value={sort.direction}
          onChange={(direction) => onChange?.({ ...sort, direction })}
        />
      </div>
      <div>
        <Button
          data-test="remove"
          icon={<Icon name="close" title={REMOVE_SORT_TITLE} />}
          size="small"
          type="text"
          onClick={onRemove}
        />
      </div>
    </div>
  );
};

const MultiSort: React.FC<MultiSortProps> = ({ sorts, columns, onChange, bannedSortColumns }) => {
  const makeOnRowChange = (idx: number) => (sort: Sort) => {
    const newSorts = [...sorts];
    newSorts[idx] = {
      ...sort,
      direction: sort.direction || 'asc',
    };
    onChange?.(newSorts);
  };
  const makeOnRowRemove = (idx: number) => () => {
    const newSorts = sorts.filter((_, cidx) => cidx !== idx);
    onChange?.(newSorts.length > 0 ? newSorts : [EMPTY_SORT]);
  };
  const addRow = () => onChange?.([...sorts, EMPTY_SORT]);
  const clearAll = () => {
    // close the popover -- happens before the onchange so the onclose handler fires first
    window.document.body.dispatchEvent(new Event('mousedown', { bubbles: true }));
    onChange?.([EMPTY_SORT]);
  };
  // replace duplicate columns with a single unspecified column for copy
  // reasons. maintain types so we can display in the select
  const [mergedColumns, typeMap] = useMemo(() => {
    const loadableTuple = columns.map((c) => {
      const columnGroups = groupBy(c, 'column');
      const fixedColumns = Object.values(columnGroups).flatMap((group) => {
        if (group.length > 1) {
          return [
            {
              ...group[0],
              type: V1ColumnType.UNSPECIFIED,
            },
          ];
        }
        return group;
      });
      const typeMap = mapValues(columnGroups, (group) => group.map((column) => column.type));
      return [fixedColumns, typeMap] as const;
    });
    return [loadableTuple.map((l) => l[0]), loadableTuple.map((l) => l[1]).getOrElse({})];
  }, [columns]);

  return (
    <div className={css.base} data-test-component="multiSort">
      <div>{SORT_MENU_TITLE}</div>
      <div className={css.rows} data-test="rows">
        {sorts.map((sort, idx) => {
          const seenColumns = sorts.slice(0, idx).map((s) => s.column);
          const columnOptions = mergedColumns.map((cols) =>
            cols.filter((c) => !seenColumns.includes(c.column)),
          );
          return (
            <MultiSortRow
              bannedSortColumns={bannedSortColumns ?? new Set()}
              columns={columnOptions}
              key={sort.column || idx}
              sort={sort}
              typeMap={typeMap}
              onChange={makeOnRowChange(idx)}
              onRemove={makeOnRowRemove(idx)}
            />
          );
        })}
      </div>
      <div className={css.actions}>
        <Button
          data-test="add"
          icon={<Icon decorative name="add" size="tiny" />}
          type="text"
          onClick={addRow}>
          {ADD_SORT_TEXT}
        </Button>
        <Button data-test="reset" type="text" onClick={clearAll}>
          {RESET_SORT_TEXT}
        </Button>
      </div>
    </div>
  );
};

const MultiSortMenu: React.FC<MultiSortProps> = ({
  sorts,
  columns,
  onChange,
  isMobile = false,
  bannedSortColumns,
}) => {
  const validSorts = sorts.filter(validSort.is);
  const onSortPopoverOpenChange = (open: boolean) => {
    if (!open) {
      onChange?.(validSorts.length > 0 ? validSorts : [EMPTY_SORT]);
    }
  };

  return (
    <Dropdown
      content={
        <MultiSort
          bannedSortColumns={bannedSortColumns}
          columns={columns}
          sorts={sorts}
          onChange={onChange}
        />
      }
      onOpenChange={onSortPopoverOpenChange}>
      <Button data-testid={SORT_MENU_BUTTON} hideChildren={isMobile} icon={<SortButtonIcon />}>
        Sort {validSorts.length ? `(${validSorts.length})` : ''}
      </Button>
    </Dropdown>
  );
};

export default MultiSortMenu;
