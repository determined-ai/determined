import { Popover } from 'antd';

import Button from 'components/kit/Button';
import Select, { Option } from 'components/kit/Select';
import { V1ColumnType, V1ProjectColumn } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import { Loadable } from 'utils/loadable';

import css from './MultiSortMenu.module.scss';

const INITIAL_SORTS = [{ column: undefined, direction: undefined }];

type DirectionType = 'asc' | 'desc';
export interface Sort {
  column?: string;
  direction?: DirectionType;
}
export type ValidSort = Required<Sort>;

export const isValidSort = (s: Sort): s is ValidSort => !!(s.column && s.direction);

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

const DirectionOptions: React.FC<DirectionOptionsProps> = ({ onChange, type, value }) => {
  const [ascText, descText] = {
    [V1ColumnType.NUMBER]: ['0 - 9', '9 - 0'],
    [V1ColumnType.TEXT]: ['A - Z', 'Z - A'],
    [V1ColumnType.DATE]: ['Oldest - Newest', 'Newest - Oldest'],
    [V1ColumnType.UNSPECIFIED]: ['Ascending', 'Descending'],
  }[type];
  return (
    <Select
      placeholder="Direction"
      value={value}
      width="100%"
      onChange={(val) => onChange(val as DirectionType)}>
      <Option value="asc">{ascText}</Option>
      <Option value="desc">{descText}</Option>
    </Select>
  );
};

const ColumnOptions: React.FC<ColumnOptionsProps> = ({ onChange, columns, value }) => {
  return (
    <Select
      loading={Loadable.isLoading(columns)}
      options={Loadable.getOrElse([], columns).map((c) => ({
        label: c.displayName || c.column,
        value: c.column,
      }))}
      placeholder="Column"
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
          danger
          icon={<Icon name="close" />}
          shape="circle"
          size="small"
          onClick={onRemove}
        />
      </div>
    </div>
  );
};

const MultiSort: React.FC<MultiSortProps> = ({ sorts, columns, onChange }) => {
  const makeOnRowChange = (idx: number) => (sort: Sort) => {
    const newSorts = [...sorts];
    newSorts[idx] = sort;
    onChange(newSorts);
  };
  const makeOnRowRemove = (idx: number) => () => {
    const newSorts = sorts.filter((_, cidx) => cidx !== idx);
    onChange(newSorts.length > 0 ? newSorts : INITIAL_SORTS);
  };
  const addRow = () => onChange([...sorts, { column: undefined, direction: undefined }]);
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
      <div>
        <Button type="link" onClick={addRow}>
          <Icon name="add-small" size="tiny" /> Add condition
        </Button>
      </div>
    </div>
  );
};
const MultiSortMenu: React.FC<MultiSortProps> = ({ sorts, columns, onChange }) => {
  const validSorts = sorts.filter(isValidSort);
  const onSortPopoverOpenChange = (open: boolean) => {
    if (!open) {
      onChange(validSorts.length > 0 ? validSorts : INITIAL_SORTS);
    }
  };

  return (
    <Popover
      content={<MultiSort columns={columns} sorts={sorts} onChange={onChange} />}
      placement="bottomRight"
      showArrow={false}
      trigger="click"
      onOpenChange={onSortPopoverOpenChange}>
      <Button>Sort {validSorts.length ? `(${validSorts.length})` : ''}</Button>
    </Popover>
  );
};

export default MultiSortMenu;
