import { Popover } from 'antd';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Form from 'components/kit/Form';
import Input, { InputRef } from 'components/kit/Input';
import Pivot from 'components/kit/Pivot';
import { V1LocationType } from 'services/api-ts-sdk';
import Spinner from 'shared/components/Spinner';
import { isEqual } from 'shared/utils/data';
import { ProjectColumn } from 'types';
import { Loadable } from 'utils/loadable';

import { defaultExperimentColumns } from './columns';

const BANNED_COLUMNS = new Set(['name']);

interface Props {
  initialVisibleColumns: string[];
  projectColumns: Loadable<ProjectColumn[]>;
  setVisibleColumns: (newColumns: string[]) => void;
}

const ColumnPickerMenu: React.FC<Props> = ({
  projectColumns,
  setVisibleColumns,
  initialVisibleColumns,
}) => {
  const [form] = Form.useForm();
  const removeBannedColumns = useCallback(
    (columns: ProjectColumn[]) => columns.filter((col) => !BANNED_COLUMNS.has(col.column)),
    [],
  );
  const [totalColumns, setTotalColumns] = useState<ProjectColumn[]>(() =>
    removeBannedColumns(Loadable.getOrElse([], projectColumns)),
  );
  const [filteredColumns, setFilteredColumns] = useState<Set<string>>(
    () =>
      new Set(removeBannedColumns(Loadable.getOrElse([], projectColumns)).map((col) => col.column)),
  );
  const [isColumnsOpen, setIsColumnsOpen] = useState(false);
  const [activeColumnTab, setActiveColumnTab] = useState<V1LocationType>(V1LocationType.EXPERIMENT);
  const searchRef = useRef<InputRef>(null);

  const columnSearch: string = Form.useWatch('column-search', form) ?? '';

  useEffect(() => {
    setTotalColumns(removeBannedColumns(Loadable.getOrElse([], projectColumns)));
  }, [projectColumns, removeBannedColumns]);

  useEffect(() => {
    const regex = new RegExp(columnSearch, 'i');
    setFilteredColumns(
      new Set(
        totalColumns
          .filter((col) => regex.test(col.displayName || col.column))
          .map((col) => col.column),
      ),
    );
  }, [columnSearch, removeBannedColumns, totalColumns]);

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

  const tabFilteredColumnsAllChecked = useMemo(() => {
    return totalColumns
      .filter((col) => filteredColumns.has(col.column) && col.location === activeColumnTab)
      .map((col) => allFormColumns[col.column])
      .every((col) => col === true);
  }, [activeColumnTab, allFormColumns, filteredColumns, totalColumns]);

  const handleShowHideAll = useCallback(() => {
    const currentTabColumns = Object.fromEntries(
      totalColumns
        .filter((col) => isEqual(col.location, activeColumnTab) && col.column in allFormColumns)
        .map((col) => [col.column, allFormColumns[col.column]]),
    );
    const filteredTabColumns: Record<string, boolean> = totalColumns
      .filter((col) => filteredColumns.has(col.column) && col.location === activeColumnTab)
      .reduce((acc, col) => ({ ...acc, [col.column]: !tabFilteredColumnsAllChecked }), {});

    form.setFieldValue(activeColumnTab, Object.assign(currentTabColumns, filteredTabColumns));
  }, [
    activeColumnTab,
    allFormColumns,
    filteredColumns,
    form,
    tabFilteredColumnsAllChecked,
    totalColumns,
  ]);

  const tabContent = useCallback(
    (tab: V1LocationType) => {
      return (
        <div>
          <Form.Item name="column-search">
            <Input allowClear placeholder="Search" ref={searchRef} />
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
              {tabFilteredColumnsAllChecked ? 'Hide' : 'Show'} all
            </Button>
            <Button type="text" onClick={handleShowSuggested}>
              Show suggested
            </Button>
          </div>
        </div>
      );
    },
    [
      filteredColumns,
      handleShowHideAll,
      handleShowSuggested,
      initialVisibleColumns,
      tabFilteredColumnsAllChecked,
      totalColumns,
    ],
  );

  const handleColumnTabChange = useCallback((activeKey: string) => {
    setActiveColumnTab(activeKey as V1LocationType);
  }, []);

  const handleOnOpenChange = useCallback((open: boolean) => {
    setIsColumnsOpen(open);
  }, []);

  useEffect(() => {
    if (!isColumnsOpen) return;
    searchRef.current?.focus();
  }, [isColumnsOpen]);

  return (
    <Popover
      content={
        <div style={{ width: '300px' }}>
          <Form form={form}>
            <Pivot
              items={[
                {
                  children: tabContent(V1LocationType.EXPERIMENT),
                  forceRender: true,
                  key: 'general',
                  label: 'General',
                },
                {
                  children: tabContent(V1LocationType.VALIDATIONS),
                  forceRender: true,
                  key: 'metrics',
                  label: 'Metrics',
                },
                {
                  children: tabContent(V1LocationType.HYPERPARAMETERS),
                  forceRender: true,
                  key: 'hyperparameters',
                  label: 'Hyperparameters',
                },
              ]}
              onChange={handleColumnTabChange}
            />
          </Form>
        </div>
      }
      placement="bottom"
      trigger="click"
      onOpenChange={handleOnOpenChange}>
      <Button>Columns</Button>
    </Popover>
  );
};

export default ColumnPickerMenu;
