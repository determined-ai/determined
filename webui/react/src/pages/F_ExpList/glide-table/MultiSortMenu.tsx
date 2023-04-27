import { Popover } from 'antd';
import * as io from 'io-ts';

import Button from 'components/kit/Button';
import Select from 'components/kit/Select';
import { V1ColumnType, V1ProjectColumn } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import { Loadable } from 'utils/loadable';

import css from './MultiSortMenu.module.scss';

const directionType = io.keyof({ asc: null, desc: null });
type DirectionType = io.TypeOf<typeof directionType>;

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
  columns: Loadable<V1ProjectColumn[]>;
  onChange: (sorts: Sort[]) => void;
}
interface MultiSortRowProps {
  sort: Sort;
  columns: Loadable<V1ProjectColumn[]>;
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
  columns: Loadable<V1ProjectColumn[]>;
  value?: string;
}

const optionsByColumnType = {
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

const DirectionOptions: React.FC<DirectionOptionsProps> = ({ onChange, type, value }) => (
  <Select
    options={optionsByColumnType[type]}
    placeholder="Select direction"
    value={value}
    width="100%"
    onChange={(val) => onChange(val as DirectionType)}
  />
);

const ColumnOptions: React.FC<ColumnOptionsProps> = ({ onChange, columns, value }) => {
  return (
    <Select
      autoFocus
      loading={Loadable.isLoading(columns)}
      options={Loadable.getOrElse([], columns).map((c) => ({
        label: c.displayName || c.column,
        value: c.column,
      }))}
      placeholder="Select column"
      value={value}
      width="100%"
      onChange={(val) => onChange(val as string)}
    />
  );
};

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
          icon={<Icon name="close" />}
          shape="circle"
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
  const addRow = () => onChange([...sorts, { column: undefined, direction: undefined }]);
  const clearAll = () => {
    onChange([EMPTY_SORT]);
    // close the popover -- set timeout to ensure it runs after the popover close handler
    setTimeout(() => {
      window.document.body.dispatchEvent(new Event('mousedown', { bubbles: true }));
    }, 5);
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
        <Button type="text" onClick={addRow}>
          <Icon name="add-small" size="tiny" /> Add sort
        </Button>
        <Button type="text" onClick={clearAll}>
          Clear all
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
      <Button>Sort {validSorts.length ? `(${validSorts.length})` : ''}</Button>
    </Popover>
  );
};

export default MultiSortMenu;
