/* eslint-disable react/jsx-sort-props */
import DataEditor, {
  CompactSelection,
  DataEditorProps,
  GridCell,
  GridCellKind,
  GridColumn,
  GridSelection,
  HeaderClickedEventArgs,
  Item,
  Rectangle,
  SizedGridColumn,
  Theme,
} from '@glideapps/glide-data-grid';
import { MenuProps } from 'antd';
import React, {
  Dispatch,
  SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useNavigate } from 'react-router';

import { paths } from 'routes/utils';
import { getColor, getInitials } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { humanReadableBytes } from 'shared/utils/string';
import usersStore from 'stores/users';
import { getStateColorCssVar } from 'themes';
import { ExperimentItem } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { getDisplayName } from 'utils/user';

import { ExperimentColumn } from '../F_ExperimentList';

import LinksCell from './custom-cells/links-cell';
import RangeCell from './custom-cells/range-cell';
import SparklineCell from './custom-cells/sparkline-cell';
import SpinnerCell from './custom-cells/spinner-cell';
import TagsCell from './custom-cells/tags-cell';
import UserProfileCell from './custom-cells/user-profile-cell';
import css from './GlideTable.module.scss';
import { placeholderMenuItems, TableActionMenu, TableActionMenuProps } from './menu';
import { MapOfIdsToColors } from './useGlasbey';
import { getDurationInEnglish, getTheme, getTimeInEnglish, headerIcons } from './utils';

const GRID_HEIGHT = 700;
const cells: DataEditorProps['customRenderers'] = [
  SparklineCell,
  TagsCell,
  UserProfileCell,
  SpinnerCell,
  RangeCell,
  LinksCell,
];

interface Props {
  colorMap: MapOfIdsToColors;
  data: ExperimentItem[];

  handleScroll?: (r: Rectangle) => void;

  sortableColumnIds: ExperimentColumn[];
  setSortableColumnIds: Dispatch<SetStateAction<ExperimentColumn[]>>;

  selectedExperimentIds: string[];
  setSelectedExperimentIds: Dispatch<SetStateAction<string[]>>;
  selectAll: boolean;
  setSelectAll: Dispatch<SetStateAction<boolean>>;
}

const STATIC_COLUMNS: ExperimentColumn[] = ['selected', 'name'];

type ColumnDef = SizedGridColumn & {
  id: ExperimentColumn;
  isNumerical?: boolean;
  renderer: (record: ExperimentItem, idx: number) => GridCell;
};

export const GlideTable: React.FC<Props> = ({
  data,
  setSelectedExperimentIds,
  sortableColumnIds,
  setSortableColumnIds,
  colorMap,
  selectAll,
  setSelectAll,
  handleScroll,
}) => {
  const gridRef = useRef(null);

  const [menuIsOpen, setMenuIsOpen] = useState(false);

  const handleMenuClose = useCallback(() => {
    setMenuIsOpen(false);
  }, []);

  const {
    ui: { darkLight },
  } = useUI();

  const [menuProps, setMenuProps] = useState<Omit<TableActionMenuProps, 'open'>>({
    handleClose: handleMenuClose,
    x: 0,
    y: 0,
  });

  const users = Loadable.map(useObservable(usersStore.getUsers()), ({ users }) => users);

  const columnIds = useMemo<ExperimentColumn[]>(
    () => [...STATIC_COLUMNS, ...sortableColumnIds],
    [sortableColumnIds],
  );
  const navigate = useNavigate();
  const bodyStyles = getComputedStyle(document.body);

  const [selection, setSelection] = React.useState<GridSelection>({
    columns: CompactSelection.empty(),
    rows: CompactSelection.empty(),
  });
  const getRowThemeOverride = React.useCallback(
    (row: number): Partial<Theme> | undefined =>
      colorMap[data[row]?.id]
        ? {
            accentColor: colorMap[data[row]?.id],
            borderColor: '#F0F0F0',
          }
        : { borderColor: '#F0F0F0' },
    [colorMap, data],
  );

  useEffect(() => {
    const selectedRowIndices = selection.rows.toArray();
    setSelectedExperimentIds((prevIds) => {
      const selectedIds = selectedRowIndices
        .map((idx) => String(data?.[idx]?.id))
        .filter((x) => x !== undefined);
      if (prevIds === selectedIds) return prevIds;
      return selectedIds;
    });
  }, [selection.rows, setSelectedExperimentIds, data]);

  const theme = getTheme(bodyStyles);

  const [columnWidths, setColumnWidths] = useState<Record<ExperimentColumn, number>>({
    archived: 80,
    checkpointCount: 74,
    checkpointSize: 74,
    description: 148,
    duration: 96,
    forkedFrom: 128,
    id: 50,
    name: 150,
    numTrials: 74,
    progress: 111,
    resourcePool: 140,
    searcherMetricValue: 74,
    searcherType: 140,
    selected: 45,
    startTime: 118,
    state: 106,
    tags: 106,
    user: 85,
  });

  const columnDefs = useMemo<Record<ExperimentColumn, ColumnDef>>(
    () => ({
      archived: {
        id: 'archived',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: String(record.archived),
          displayData: record.archived ? '📦' : '',
          kind: GridCellKind.Text,
        }),
        title: 'Archived',
        width: columnWidths.archived,
      },
      checkpointCount: {
        id: 'checkpointCount',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: Number(record.checkpointCount),
          displayData: String(record.checkpointCount),
          kind: GridCellKind.Number,
        }),
        title: 'Checkpoint Count',
        width: columnWidths.checkpointCount,
      },
      checkpointSize: {
        id: 'checkpointSize',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: record.checkpointSize ? humanReadableBytes(record.checkpointSize) : '',
          displayData: record.checkpointSize ? humanReadableBytes(record.checkpointSize) : '',
          kind: GridCellKind.Text,
        }),
        title: 'Checkpoint Size',
        width: columnWidths.checkpointSize,
      },
      description: {
        id: 'description',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: String(record.description),
          displayData: String(record.description),
          kind: GridCellKind.Text,
        }),
        title: 'Description',
        width: columnWidths.description,
      },
      duration: {
        id: 'duration',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: `${getDurationInEnglish(record)}`,
          displayData: `${getDurationInEnglish(record)}`,
          kind: GridCellKind.Text,
        }),
        title: 'Duration',
        width: columnWidths.duration,
      },
      forkedFrom: {
        id: 'forkedFrom',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          copyData: String(record.forkedFrom ?? ''),
          data: {
            kind: 'links-cell',
            links:
              record.forkedFrom !== undefined
                ? [
                    {
                      onClick: () =>
                        record.forkedFrom && navigate(paths.experimentDetails(record.forkedFrom)),
                      title: String(record.forkedFrom ?? ''),
                    },
                  ]
                : [],
            navigateOn: 'click',
            underlineOffset: 6,
          },
          kind: GridCellKind.Custom,
          readonly: true,
        }),
        title: 'Forked From',
        width: columnWidths.forkedFrom,
      },
      id: {
        id: 'id',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          copyData: String(record.id),
          data: {
            kind: 'links-cell',
            links: [
              {
                onClick: () => navigate(paths.experimentDetails(record.id)),
                title: String(record.id),
              },
            ],
            navigateOn: 'click',
            underlineOffset: 6,
          },
          kind: GridCellKind.Custom,
          readonly: true,
        }),
        title: 'ID',
        width: columnWidths.id,
      },
      name: {
        id: 'name',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          copyData: String(record.name),
          data: {
            kind: 'links-cell',
            links: [
              {
                onClick: () => navigate(paths.experimentDetails(record.id)),
                title: String(record.name),
              },
            ],
            navigateOn: 'click',
            underlineOffset: 6,
          },
          kind: GridCellKind.Custom,
          readonly: true,
        }),
        themeOverride: { horizontalBorderColor: '#225588' },
        title: 'Name',
        width: columnWidths.name,
      },
      numTrials: {
        id: 'numTrials',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: record.numTrials,
          displayData: String(record.numTrials),
          kind: GridCellKind.Number,
        }),
        title: 'Trials',
        width: columnWidths.numTrials,
      },
      progress: {
        id: 'progress',
        renderer: (record: ExperimentItem) => {
          return (record.progress ?? 0) > 0
            ? {
                allowOverlay: false,
                copyData: String(record.progress ?? 0),
                data: {
                  color: bodyStyles.getPropertyValue(
                    getStateColorCssVar(record.state).slice(4, -1),
                  ),
                  kind: 'range-cell',
                  max: 1,
                  min: 0,
                  step: 1,
                  value: record.progress ?? 0,
                },
                kind: GridCellKind.Custom,
              }
            : {
                allowOverlay: false,
                data: '',
                displayData: '',
                kind: GridCellKind.Text,
              };
        },
        title: 'Progress',
        width: columnWidths.progress,
      },
      resourcePool: {
        id: 'resourcePool',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: String(record.resourcePool),
          displayData: String(record.resourcePool),
          kind: GridCellKind.Text,
        }),
        title: 'Resource Pool',
        width: columnWidths.resourcePool,
      },
      searcherMetricValue: {
        id: 'searcherMetricValue',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: String(record.searcherMetricValue ?? ''),
          displayData: String(record.searcherMetricValue ?? ''),
          kind: GridCellKind.Text,
        }),
        title: 'Searcher Metric Values',
        width: columnWidths.searcherMetricValue,
      },
      searcherType: {
        id: 'searcherType',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: String(record.searcherType),
          displayData: String(record.searcherType),
          kind: GridCellKind.Text,
        }),
        title: 'Searcher Type',
        width: columnWidths.searcherType,
      },
      selected: {
        icon: 'selected',
        id: 'selected',
        renderer: (record: ExperimentItem, idx) => ({
          allowOverlay: false,
          contentAlign: 'left',
          data: selectAll || selection.rows.hasIndex(idx),
          // disabled: selectAll,
          kind: GridCellKind.Boolean,
        }),
        themeOverride: { cellHorizontalPadding: 13, headerIconSize: 30 },
        title: '',
        width: columnWidths.selected,
      },
      startTime: {
        id: 'startTime',
        isNumerical: true,
        renderer: (record: ExperimentItem) => ({
          allowOverlay: false,
          data: getTimeInEnglish(new Date(record.startTime)),
          displayData: getTimeInEnglish(new Date(record.startTime)),
          kind: GridCellKind.Text,
        }),
        title: 'Start Time',
        width: columnWidths.startTime,
      },
      state: {
        id: 'state',
        renderer: (record: ExperimentItem) => ({
          allowAdd: false,
          allowOverlay: true,
          copyData: record.state.toLocaleLowerCase(),
          data: [],
          kind: GridCellKind.Image,
        }),
        title: 'State',
        width: columnWidths.state,
      },
      tags: {
        id: 'tags',
        renderer: (record: ExperimentItem) => ({
          allowOverlay: true,
          copyData: record['labels'].join(', '),
          data: {
            kind: 'tags-cell',
            possibleTags: [],
            readonly: true,
            tags: record['labels'],
          },
          kind: GridCellKind.Custom,
        }),
        title: 'Tags',
        width: columnWidths.tags,
      },
      user: {
        id: 'user',
        renderer: (record: ExperimentItem) => {
          const displayName = Loadable.match(users, {
            Loaded: (users) => getDisplayName(users?.find((u) => u.id === record.userId)),
            NotLoaded: () => undefined,
          });
          return {
            allowOverlay: true,
            copyData: String(record.userId),
            data: {
              image: undefined,
              initials: getInitials(displayName),
              kind: 'user-profile-cell',
              tint: getColor(displayName, darkLight),
            },
            kind: GridCellKind.Custom,
          };
        },

        title: 'User',
        width: columnWidths.user,
      },
    }),
    [navigate, selectAll, selection.rows, columnWidths, users, darkLight],
  );

  const onColumnResize = useCallback((column: GridColumn, width: number) => {
    const columnId = column.id;
    if (columnId === undefined || columnId === 'selected') return;
    setColumnWidths((prevWidths) => {
      const prevWidth = prevWidths[columnId as ExperimentColumn];
      if (width === prevWidth) return prevWidths;
      return { ...prevWidths, [columnId]: width };
    });
  }, []);

  const onColumnResizeEnd = useCallback(() => {
    // presumably update the settings, but maybe have a different API
    // like Record<ColumnName, width>
  }, []);

  const onHeaderClicked = React.useCallback(
    (col: number, args: HeaderClickedEventArgs) => {
      const { bounds } = args;
      const columnId = columnIds[col];

      if (columnId === 'selected') {
        setSelectAll((prev) => !prev);
        return;
      }

      const items: MenuProps['items'] = placeholderMenuItems;
      const x = bounds.x;
      const y = bounds.y + bounds.height;
      setMenuProps((prev) => ({ ...prev, items, title: `${columnId} menu`, x, y }));
      setMenuIsOpen(true);
    },
    [columnIds, setSelectAll],
  );

  const getContent = React.useCallback(
    (cell: Item): GridCell => {
      const [colIdx, rowIdx] = cell;
      const columnId = columnIds[colIdx];
      return columnDefs[columnId].renderer(data[rowIdx], rowIdx);
    },
    [data, columnIds, columnDefs],
  );

  const handleGridSelectionChange = useCallback((newSelection: GridSelection) => {
    const [, row] = newSelection.current?.cell ?? [undefined, undefined];
    if (row === undefined) return;
    setSelection(({ rows }: GridSelection) => ({
      columns: CompactSelection.empty(),
      rows: rows.hasIndex(row) ? rows.remove(row) : rows.add(row),
    }));
  }, []);

  const onColumnMoved = useCallback(
    (_startIndex: number, _endIndex: number): void => {
      const startIndex = _startIndex - STATIC_COLUMNS.length;
      const endIndex = Math.max(_endIndex - STATIC_COLUMNS.length, 0);
      if (startIndex > -1) {
        setSortableColumnIds((prevCols) => {
          const newCols = [...prevCols];
          const [toMove] = newCols.splice(startIndex, 1);
          newCols.splice(endIndex, 0, toMove);
          return newCols;
        });
      }
    },
    [setSortableColumnIds],
  );

  const dataGridColumns = useMemo(
    () => columnIds.map((columnName) => columnDefs[columnName as ExperimentColumn]) as GridColumn[],
    [columnIds, columnDefs],
  );

  return (
    <div className={css.base}>
      <DataEditor
        headerIcons={headerIcons}
        customRenderers={cells}
        columns={dataGridColumns}
        freezeColumns={2}
        getCellContent={getContent}
        gridSelection={selection}
        height={GRID_HEIGHT}
        ref={gridRef}
        rows={data.length}
        smoothScrollX
        smoothScrollY
        width="98%"
        onColumnMoved={onColumnMoved}
        onColumnResize={onColumnResize}
        onColumnResizeEnd={onColumnResizeEnd}
        onGridSelectionChange={handleGridSelectionChange}
        getRowThemeOverride={getRowThemeOverride}
        onVisibleRegionChanged={handleScroll}
        theme={theme}
        onHeaderClicked={onHeaderClicked}
        // these might come in handy
        // onCellClicked={onCellClicked}
        // onItemHovered={onItemHovered}
        // onHeaderContextMenu={onHeaderContextMenu}
        // onCellContextMenu={onCellContextMenu}
      />
      <TableActionMenu {...menuProps} open={menuIsOpen} />
    </div>
  );
};

export default GlideTable;
