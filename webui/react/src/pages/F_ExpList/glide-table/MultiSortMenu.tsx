import { Col, Row } from 'antd';

import Button from 'components/kit/Button';
import Select, { Option } from 'components/kit/Select';
import { V1ColumnType, V1ProjectColumn } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import { Loadable } from 'utils/loadable';

import css from './MultiSortMenu.module.scss';

type DirectionType = 'asc' | 'desc';
export interface Sort {
  column?: string;
  direction?: DirectionType;
}
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
    [V1ColumnType.NUMBER]: ['1 - 9', '9 - 1'],
    [V1ColumnType.TEXT]: ['A - Z', 'Z - A'],
    // TODO: Choose a less cute end date -- current one's specificity might give
    // the impression that we're looking at values
    [V1ColumnType.DATE]: [
      '1970/01/01 00:00 - 2038/19/01 03:14',
      '2038/19/01 03:14 - 1970/01/01 00:00',
    ],
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
        label: c.displayName,
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
    <Row align="middle" gutter={8}>
      <Col flex="auto">
        <ColumnOptions
          columns={columns}
          value={sort.column}
          onChange={(column) => onChange({ ...sort, column })}
        />
      </Col>
      <Col flex="auto">
        <DirectionOptions
          type={valueType}
          value={sort.direction}
          onChange={(direction) => onChange({ ...sort, direction })}
        />
      </Col>
      <Col flex="none">
        <Button
          danger
          icon={<Icon name="close" />}
          shape="circle"
          size="small"
          onClick={onRemove}
        />
      </Col>
    </Row>
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
    onChange(newSorts.length > 0 ? newSorts : [{ column: undefined, direction: undefined }]);
  };
  const addRow = () => onChange([...sorts, { column: undefined, direction: undefined }]);
  return (
    <div className={css.base}>
      <div>Sort by</div>
      <div>
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

export default MultiSort;
