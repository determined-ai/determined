import { Popover } from 'antd';
import React, { useCallback, useEffect, useMemo } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Form from 'components/kit/Form';
import { FormInstance } from 'components/kit/Form';
import Input from 'components/kit/Input';
import Pivot from 'components/kit/Pivot';
import { V1LocationType } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { isEqual } from 'shared/utils/data';
import { ProjectColumn } from 'types';
import { Loadable } from 'utils/loadable';

import { defaultExperimentColumns } from './columns';

const BANNED_COLUMNS = new Set(['name']);

const removeBannedColumns = (columns: ProjectColumn[]) =>
  columns.filter((col) => !BANNED_COLUMNS.has(col.column));

const locationLabelMap = {
  [V1LocationType.EXPERIMENT]: 'General',
  [V1LocationType.VALIDATIONS]: 'Metrics',
  [V1LocationType.HYPERPARAMETERS]: 'Hyperparameters',
} as const;

interface ColumnMenuProps {
  initialVisibleColumns: string[];
  projectColumns: Loadable<ProjectColumn[]>;
  setVisibleColumns: (newColumns: string[]) => void;
}

interface ColumnTabProps {
  allFormColumns: Record<string, boolean>;
  filteredColumns: Set<string>;
  form: FormInstance;
  handleShowSuggested: () => void;
  initialVisibleColumns: string[];
  tab: V1LocationType;
  totalColumns: ProjectColumn[];
}

const ColumnPickerTab: React.FC<ColumnTabProps> = ({
  allFormColumns,
  filteredColumns,
  form,
  handleShowSuggested,
  initialVisibleColumns,
  tab,
  totalColumns,
}) => {
  const allFilteredColumnsChecked = useMemo(() => {
    return totalColumns
      .filter((col) => filteredColumns.has(col.column) && col.location === tab)
      .map((col) => allFormColumns[col.column])
      .every((col) => col === true);
  }, [tab, allFormColumns, filteredColumns, totalColumns]);

  const handleShowHideAll = useCallback(() => {
    const currentTabColumns = Object.fromEntries(
      totalColumns
        .filter((col) => isEqual(col.location, tab) && col.column in allFormColumns)
        .map((col) => [col.column, allFormColumns[col.column]]),
    );
    const filteredTabColumns: Record<string, boolean> = totalColumns
      .filter((col) => filteredColumns.has(col.column) && col.location === tab)
      .reduce((acc, col) => ({ ...acc, [col.column]: !allFilteredColumnsChecked }), {});

    form.setFieldValue(tab, Object.assign(currentTabColumns, filteredTabColumns));
  }, [tab, allFormColumns, filteredColumns, form, allFilteredColumnsChecked, totalColumns]);

  return (
    <div>
      <Form.Item name="column-search">
        <Input allowClear autoFocus placeholder="Search" />
      </Form.Item>
      {totalColumns.length !== 0 ? (
        <div style={{ maxHeight: 360, overflow: 'hidden auto' }}>
          {totalColumns
            .filter((column) => column.location === tab)
            .map((column) => (
              <Form.Item
                hidden={!filteredColumns.has(column.column)}
                initialValue={initialVisibleColumns.includes(column.column)}
                key={column.column}
                name={[tab, column.column]}
                valuePropName="checked">
                <Checkbox>{column.displayName || column.column}</Checkbox>
              </Form.Item>
            ))}
        </div>
      ) : (
        <Spinner />
      )}
      <div style={{ display: 'flex', justifyContent: 'space-between' }}>
        <Button type="text" onClick={handleShowHideAll}>
          {allFilteredColumnsChecked ? 'Hide' : 'Show'} all
        </Button>
        <Button type="text" onClick={handleShowSuggested}>
          Show suggested
        </Button>
      </div>
    </div>
  );
};

const ColumnPickerMenu: React.FC<ColumnMenuProps> = ({
  projectColumns,
  setVisibleColumns,
  initialVisibleColumns,
}) => {
  const [form] = Form.useForm();

  const columnSearch: string = Form.useWatch('column-search', form) ?? '';

  const totalColumns = useMemo(
    () => removeBannedColumns(Loadable.getOrElse([], projectColumns)),
    [projectColumns],
  );

  const filteredColumns = useMemo(() => {
    const regex = new RegExp(columnSearch, 'i');
    return new Set(
      totalColumns
        .filter((col) => regex.test(col.displayName || col.column))
        .map((col) => col.column),
    );
  }, [columnSearch, totalColumns]);

  const generalColumns: Record<string, boolean> = Form.useWatch(V1LocationType.EXPERIMENT, form);
  const hyperparametersColumns: Record<string, boolean> = Form.useWatch(
    V1LocationType.HYPERPARAMETERS,
    form,
  );
  const metricsColumns: Record<string, boolean> = Form.useWatch(V1LocationType.VALIDATIONS, form);

  const allFormColumns = useMemo(
    () => ({ ...generalColumns, ...hyperparametersColumns, ...metricsColumns }),
    [generalColumns, hyperparametersColumns, metricsColumns],
  );

  useEffect(() => {
    if (Object.keys(allFormColumns).length === 0) return;
    /* eslint-disable @typescript-eslint/no-unused-vars */
    setVisibleColumns(
      Object.entries(allFormColumns)
        .filter(([_, checked]) => checked)
        .map(([column, _]) => column),
    );
    /* eslint-enable @typescript-eslint/no-unused-vars */
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allFormColumns, setVisibleColumns]);

  const handleShowSuggested = useCallback(() => {
    const defaultCols: Set<string> = new Set(defaultExperimentColumns);

    const newGeneral = { ...generalColumns };
    for (const col of Object.keys(newGeneral)) {
      newGeneral[col] = defaultCols.has(col);
    }
    form.setFieldValue(V1LocationType.EXPERIMENT, newGeneral);

    const newHyperparameters = { ...hyperparametersColumns };
    for (const col of Object.keys(newHyperparameters)) {
      newHyperparameters[col] = defaultCols.has(col);
    }
    form.setFieldValue(V1LocationType.HYPERPARAMETERS, newHyperparameters);

    const newMetrics = { ...metricsColumns };
    for (const col of Object.keys(newMetrics)) {
      newMetrics[col] = defaultCols.has(col);
    }
    form.setFieldValue(V1LocationType.VALIDATIONS, newMetrics);
  }, [form, generalColumns, hyperparametersColumns, metricsColumns]);

  return (
    <Popover
      content={
        <div style={{ width: '300px' }}>
          <Form form={form}>
            <Pivot
              items={[
                V1LocationType.EXPERIMENT,
                V1LocationType.VALIDATIONS,
                V1LocationType.HYPERPARAMETERS,
              ].map((tab) => ({
                children: (
                  <ColumnPickerTab
                    allFormColumns={allFormColumns}
                    filteredColumns={filteredColumns}
                    form={form}
                    handleShowSuggested={handleShowSuggested}
                    initialVisibleColumns={initialVisibleColumns}
                    tab={tab}
                    totalColumns={totalColumns}
                  />
                ),
                forceRender: true,
                key: tab,
                label: locationLabelMap[tab],
              }))}
            />
          </Form>
        </div>
      }
      placement="bottom"
      trigger="click">
      <Button>Columns</Button>
    </Popover>
  );
};

export default ColumnPickerMenu;
