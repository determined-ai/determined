import { Card as AntDCard, Space } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';

import Grid from 'components/Grid';
import Accordion from 'components/kit/Accordion';
import Avatar from 'components/kit/Avatar';
import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import Card from 'components/kit/Card';
import Checkbox from 'components/kit/Checkbox';
import ClipboardButton from 'components/kit/ClipboardButton';
import CodeEditor from 'components/kit/CodeEditor';
import { Column, Columns } from 'components/kit/Columns';
import DatePicker from 'components/kit/DatePicker';
import Drawer from 'components/kit/Drawer';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Form from 'components/kit/Form';
import Icon, { IconNameArray, IconSizeArray } from 'components/kit/Icon';
import InlineForm from 'components/kit/InlineForm';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import InputSearch from 'components/kit/InputSearch';
import InputShortcut, { KeyboardShortcut } from 'components/kit/InputShortcut';
import { TypographySize } from 'components/kit/internal/fonts';
import { MetricType, Note, Serie, ValueOf, XAxisDomain } from 'components/kit/internal/types';
import { LineChart } from 'components/kit/LineChart';
import { useChartGrid } from 'components/kit/LineChart/useChartGrid';
import LogViewer from 'components/kit/LogViewer/LogViewer';
import Message from 'components/kit/Message';
import { Modal, useModal } from 'components/kit/Modal';
import Nameplate from 'components/kit/Nameplate';
import Notes, { Props as NotesProps } from 'components/kit/Notes';
import Pagination from 'components/kit/Pagination';
import Pivot from 'components/kit/Pivot';
import Select, { Option } from 'components/kit/Select';
import Spinner from 'components/kit/Spinner';
import useUI from 'components/kit/Theme';
import { makeToast } from 'components/kit/Toast';
import Toggle from 'components/kit/Toggle';
import Tooltip from 'components/kit/Tooltip';
import Header from 'components/kit/Typography/Header';
import Paragraph from 'components/kit/Typography/Paragraph';
import useConfirm, { voidPromiseFn } from 'components/kit/useConfirm';
import { useTags } from 'components/kit/useTags';
import { Loadable, Loaded, NotLoaded } from 'components/kit/utils/loadable';
import Label from 'components/Label';
import KitLink from 'components/Link';
import Logo from 'components/Logo';
import Page from 'components/Page';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import ThemeToggle from 'components/ThemeToggle';
import { drawPointsPlugin } from 'components/UPlot/UPlotChart/drawPointsPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin';
import { CheckpointsDict } from 'pages/TrialDetails/TrialDetailsMetrics';
import { serverAddress } from 'routes/utils';
import { V1LogLevel } from 'services/api-ts-sdk';
import { mapV1LogsResponse } from 'services/decoder';
import { BrandingType } from 'stores/determinedInfo';
import {
  Background,
  Brand,
  Float,
  Interactive,
  Overlay,
  Stage,
  Status,
  Surface,
} from 'utils/colors';
import handleError from 'utils/error';
import loremIpsum, { loremIpsumSentence } from 'utils/loremIpsum';
import { noOp } from 'utils/service';

import css from './DesignKit.module.scss';

const ComponentTitles = {
  Accordion: 'Accordion',
  Avatar: 'Avatar',
  Breadcrumbs: 'Breadcrumbs',
  Buttons: 'Buttons',
  Cards: 'Cards',
  Charts: 'Charts',
  Checkboxes: 'Checkboxes',
  ClipboardButton: 'ClipboardButton',
  CodeEditor: 'CodeEditor',
  Color: 'Color',
  Columns: 'Columns',
  DatePicker: 'DatePicker',
  Drawer: 'Drawer',
  Dropdown: 'Dropdown',
  Form: 'Form',
  Icons: 'Icons',
  InlineForm: 'InlineForm',
  Input: 'Input',
  InputNumber: 'InputNumber',
  InputSearch: 'InputSearch',
  InputShortcut: 'InputShortcut',
  Lists: 'Lists (tables)',
  LogViewer: 'LogViewer',
  Message: 'Message',
  Modals: 'Modals',
  Nameplate: 'Nameplate',
  Notes: 'Notes',
  Pagination: 'Pagination',
  Pivot: 'Pivot',
  Select: 'Select',
  Spinner: 'Spinner',
  Tags: 'Tags',
  Theme: 'Theme',
  Toast: 'Toast',
  Toggle: 'Toggle',
  Tooltips: 'Tooltips',
  Typography: 'Typography',
} as const;

type ComponentNames = ValueOf<typeof ComponentTitles>;
type ComponentIds = keyof typeof ComponentTitles;

const componentOrder = Object.entries(ComponentTitles)
  .sort((pair1, pair2) => pair1[1].localeCompare(pair2[1]))
  .map((pair) => pair[0] as keyof typeof ComponentTitles);

interface Props {
  children?: React.ReactNode;
  id: ComponentIds;
  title: ComponentNames;
}

const ComponentSection: React.FC<Props> = ({ children, id, title }: Props): JSX.Element => {
  return (
    <article>
      <h3 id={id}>{title}</h3>
      {children}
    </article>
  );
};

const ButtonsSection: React.FC = () => {
  const menu: MenuItem[] = [
    { key: 'start', label: 'Start' },
    { key: 'stop', label: 'Stop' },
  ];
  return (
    <ComponentSection id="Buttons" title="Buttons">
      <AntDCard>
        <p>
          <code>{'<Button>'}</code>s give people a way to trigger an action. They&apos;re typically
          found in forms, dialog panels, and dialogs. Some buttons are specialized for particular
          tasks, such as navigation, repeated actions, or presenting menus.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            For dialog boxes and panels, where people are moving through a sequence of screens,
            right-align buttons with the container.
          </li>
          <li>For single-page forms and focused tasks, left-align buttons with the container.</li>
          <li>
            Always place the primary button on the left, the secondary button just to the right of
            it.
          </li>
          <li>
            Show only one primary button that inherits theme color at rest state. If there are more
            than two buttons with equal priority, all buttons should have neutral backgrounds.
          </li>
          <li>
            Don&apos;t use a button to navigate to another place; use a link instead. The exception
            is in a wizard where &quot;Back&quot; and &quot;Next&quot; buttons may be used.
          </li>
          <li>
            Don&apos;t place the default focus on a button that destroys data. Instead, place the
            default focus on the button that performs the &quot;safe act&quot; and retains the
            content (such as &quot;Save&quot;) or cancels the action (such as &quot;Cancel&quot;).
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            Make sure it&apos;s clear what will happen when people interact with the button. Be
            concise; usually a single verb is best. Include a noun if there is any room for
            interpretation about what the verb means. For example, &quot;Delete folder&quot; or
            &quot;Create account&quot;.
          </li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>Always enable the user to navigate to focus on buttons using their keyboard.</li>
          <li>Buttons need to have accessible naming.</li>
          <li>Aria- and roles need to have consistent (non-generic) attributes.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default Button variations</strong>
        Transparent background, solid border
        <Space>
          <Button>Default</Button>
          <Button danger>Danger</Button>
          <Button disabled>Disabled</Button>
          <Button loading>Loading</Button>
          <Button selected>Selected</Button>
        </Space>
        <hr />
        <strong>Primary Button variations</strong>
        Solid background, no border
        <Space>
          <Button type="primary">Primary</Button>
          <Button danger type="primary">
            Danger
          </Button>
          <Button disabled type="primary">
            Disabled
          </Button>
          <Button loading type="primary">
            Loading
          </Button>
          <Button selected type="primary">
            Selected
          </Button>
        </Space>
        <hr />
        <strong>Text Button variations</strong>
        Transparent background, no border
        <Space>
          <Button type="text">Text</Button>
          <Button danger type="text">
            Danger
          </Button>
          <Button disabled type="text">
            Disabled
          </Button>
          <Button loading type="text">
            Loading
          </Button>
          <Button selected type="text">
            Selected
          </Button>
        </Space>
        <hr />
        <strong>Dashed Button variations</strong>
        Transparent background, dashed border
        <Space>
          <Button type="dashed">Dashed</Button>
          <Button danger type="dashed">
            Danger
          </Button>
          <Button disabled type="dashed">
            Disabled
          </Button>
          <Button loading type="dashed">
            Loading
          </Button>
          <Button selected type="dashed">
            Selected
          </Button>
        </Space>
        <hr />
        <strong>Full-width buttons</strong>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Button block>Default</Button>
          <Button block type="primary">
            Primary
          </Button>
          <Button block type="text">
            Text
          </Button>
          <Button block type="dashed">
            Dashed
          </Button>
        </Space>
        <hr />
        <strong>Sizes</strong>
        <Space>
          <Button size="large">Large</Button>
          <Button size="middle">Middle</Button>
          <Button size="small">Small</Button>
        </Space>
        <hr />
        <strong>With icon</strong>
        With Icon
        <Space>
          <Button icon={<Icon name="panel" title="compare" />} />
          <Button icon={<Icon name="panel" title="compare" />}>SVG icon</Button>
          <Button icon={<Icon name="power" title="power" />} />
          <Button icon={<Icon name="power" title="power" />}>SVG icon</Button>
        </Space>
        As Dropdown trigger with Icon
        <Space>
          <Dropdown menu={menu}>
            <Button icon={<Icon name="power" title="power" />} />
          </Dropdown>
          <Dropdown menu={menu}>
            <Button icon={<Icon name="power" title="power" />}>SVG icon</Button>
          </Dropdown>
          <Dropdown menu={menu}>
            <Button icon={<Icon name="play" size="large" title="Play" />} />
          </Dropdown>
          <Dropdown menu={menu}>
            <Button icon={<Icon name="play" size="large" title="Play" />}>Font icon</Button>
          </Dropdown>
        </Space>
        With icon and text displayed in a column
        <Space>
          <Button column icon={<Icon name="power" title="power" />} size="small">
            Column Small
          </Button>
          <Button column icon={<Icon name="power" title="power" />} size="middle">
            Column Middle
          </Button>
          <Button column icon={<Icon name="power" title="power" />} size="large">
            Column Large
          </Button>
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const SelectSection: React.FC = () => {
  const [multiSelectValues, setMultiSelectValues] = useState<SelectValue>();
  const [clearableSelectValues, setClearableSelectValues] = useState<SelectValue>();
  const [sortedSelectValues, setSortedSelectValues] = useState<SelectValue>();

  return (
    <ComponentSection id="Select" title="Select">
      <AntDCard>
        <p>
          A Select (<code>{'<Select>'}</code>) combines a text field and a dropdown giving people a
          way to select an option from a list or enter their own choice.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Use a select when there are multiple choices that can be collapsed under one title, when
            the list of items is long, or when space is constrained.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use single words or shortened statements as options.</li>
          <li>Don&apos;t use punctuation at the end of options.</li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>
            Select dropdowns render in their own layer by default to ensure they are not clipped by
            containers with overflow: hidden or overflow: scroll. This causes extra difficulty for
            people who use screen readers, so we recommend rendering the ComboBox options dropdown
            inline unless they are in overflow containers.
          </li>
        </ul>
        <strong>Truncation</strong>
        <ul>
          <li>
            By default, the Select truncates option text instead of wrapping to a new line. Because
            this can lose meaningful information, it is recommended to adjust styles to wrap the
            option text.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default Select</strong>
        <Select
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Select"
        />
        <strong>Variations</strong>
        <strong>Loading Select</strong>
        <Select
          loading
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Select"
        />
        <strong>Select with default value</strong>
        <Select
          defaultValue={2}
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
        />
        <strong>Select with label</strong>
        <Select
          label="Select Label"
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Select"
        />
        <strong>Select without placeholder</strong>
        <Select
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
        />
        <strong>Disabled Select</strong>
        <Select
          defaultValue="disabled"
          disabled
          options={[{ label: 'Disabled', value: 'disabled' }]}
        />
        <strong>Select without search</strong>
        <Select
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Nonsearcahble Select"
          searchable={false}
        />
        <strong>Multiple Select with tags</strong>
        <Select
          mode="multiple"
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Select Tags"
          width={300}
        />
        <strong>Multiple Select with tags disabled</strong>
        <Select
          disableTags
          mode="multiple"
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          placeholder="Select Multiple"
          value={multiSelectValues}
          width={150}
          onChange={(value) => setMultiSelectValues(value)}
        />
        <strong>Select with tags and custom search</strong>
        <Select
          filterOption={(input, option) =>
            !!(option?.label && option.label.toString().includes(input) === true)
          }
          mode="multiple"
          options={[
            { label: 'Case 1', value: 1 },
            { label: 'Case 2', value: 2 },
            { label: 'Case 3', value: 3 },
          ]}
          placeholder="Case-sensitive Search"
          width={300}
        />
        <strong>Select with sorted search</strong>
        <Select
          disableTags
          filterOption={(input, option) =>
            (option?.label?.toString() ?? '').toLowerCase().includes(input.toLowerCase())
          }
          filterSort={(a, b) => ((a?.label ? a.label : 0) > (b?.label ? b?.label : 0) ? 1 : -1)}
          mode="multiple"
          options={[
            { label: 'Am', value: 1 },
            { label: 'Az', value: 2 },
            { label: 'Ac', value: 3 },
            { label: 'Aa', value: 4 },
          ]}
          placeholder="Search"
          value={sortedSelectValues}
          width={120}
          onChange={(value) => setSortedSelectValues(value)}
        />
        <strong>Clearable Select</strong>
        <Select
          allowClear
          disableTags
          mode="multiple"
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          value={clearableSelectValues}
          width={130}
          onChange={(value) => setClearableSelectValues(value)}
        />
        <strong>Responsive Select with large width defined</strong>
        <Select
          disableTags
          options={[
            { label: 'Option 1', value: 1 },
            { label: 'Option 2', value: 2 },
            { label: 'Option 3', value: 3 },
          ]}
          width={999999}
        />
        <span>
          Also see <a href={`#${ComponentTitles.Form}`}>Form</a> for form-specific variations
        </span>
      </AntDCard>
    </ComponentSection>
  );
};

const ThemeSection: React.FC = () => (
  <ComponentSection id="Theme" title="Theme">
    <AntDCard>
      <p>
        <code>{'<UIProvider>'}</code> is part of the UI kit, it is responsible for handling all
        UI/theme related state, such as dark/light theme setup. It takes an optional{' '}
        <code>{'branding'}</code> prop for adjusting branding specific theme/colors.
      </p>
      <p>
        Besides the <code>{'<UIProvider>'}</code>, there are a few other helpers that can be used
        from withing the UI kit.
        <ul>
          <li>
            <code>{'useUI'}</code>, a custom hook for setting th new state for theme, mode and other
            UI-related functionalities.
          </li>
          <li>
            helper types, such as <code>{'DarkLight'}</code>.
          </li>
          <li>
            helper functions, such as <code>{'getCssVar'}</code>.
          </li>
        </ul>
      </p>
    </AntDCard>
  </ComponentSection>
);

const ChartsSection: React.FC = () => {
  const [line1Data, setLine1Data] = useState<[number, number][]>([
    [0, -2],
    [2, 7],
    [4, 15],
    [6, 35],
    [9, 22],
    [10, 76],
    [18, 1],
    [19, 89],
  ]);
  const [line2Data, setLine2Data] = useState<[number, number][]>([
    [1, 15],
    [2, 10.123456789],
    [2.5, 22],
    [3, 10.3909],
    [3.25, 19],
    [3.75, 4],
    [4, 12],
  ]);
  const [timer, setTimer] = useState(line1Data.length);
  useEffect(() => {
    let timeout: NodeJS.Timer | void;
    if (timer <= line1Data.length) {
      timeout = setTimeout(() => setTimer((t) => t + 1), 2000);
    }
    return () => timeout && clearTimeout(timeout);
  }, [timer, line1Data]);

  const randomizeLineData = useCallback(() => {
    setLine1Data([
      [0, -2],
      [2, Math.random() * 12],
      [4, 15],
      [6, Math.random() * 60],
      [9, Math.random() * 40],
      [10, Math.random() * 76],
      [18, Math.random() * 80],
      [19, 89],
    ]);
    setLine2Data([
      [1, 15],
      [2, 10.123456789],
      [2.5, Math.random() * 22],
      [3, 10.3909],
      [3.25, 19],
      [3.75, 4],
      [4, 12],
    ]);
  }, []);
  const streamLineData = useCallback(() => setTimer(1), []);

  const line1BatchesDataStreamed = useMemo(() => line1Data.slice(0, timer), [timer, line1Data]);
  const line2BatchesDataStreamed = useMemo(() => line2Data.slice(0, timer), [timer, line2Data]);

  const line1: Serie = {
    color: '#009BDE',
    data: {
      [XAxisDomain.Batches]: line1BatchesDataStreamed,
      [XAxisDomain.Time]: [],
    },
    metricType: MetricType.Training,
    name: 'Line',
  };

  const stampToNum = (tstamp: string): number => new Date(tstamp).getTime() / 1000;

  const line2: Serie = {
    data: {
      [XAxisDomain.Batches]: line2BatchesDataStreamed,
      [XAxisDomain.Time]: [
        [stampToNum('2023-01-05T01:00:00Z'), 15],
        [stampToNum('2023-01-05T02:12:34.56789Z'), 10.123456789],
        [stampToNum('2023-01-05T02:30:00Z'), 22],
        [stampToNum('2023-01-05T03:15:00Z'), 15],
        [stampToNum('2023-01-05T04:02:06Z'), 12],
      ],
    },
    metricType: MetricType.Validation,
    name: 'Line',
  };

  const line3: Serie = {
    data: {
      [XAxisDomain.Time]: [
        [stampToNum('2023-01-05T01:00:00Z'), 12],
        [stampToNum('2023-01-05T02:00:00Z'), 5],
        [stampToNum('2023-01-05T02:30:00Z'), 2],
        [stampToNum('2023-01-05T03:00:00Z'), 10.123456789],
        [stampToNum('2023-01-05T04:00:00Z'), 4],
      ],
    },
    metricType: MetricType.Validation,
    name: 'Alt-Line',
  };

  const zeroline: Serie = {
    color: '#009BDE',
    data: {
      [XAxisDomain.Batches]: [[0, 1]],
      [XAxisDomain.Time]: [],
    },
    metricType: MetricType.Training,
    name: 'Line',
  };

  const checkpointsDict: CheckpointsDict = {
    2: {
      endTime: '2023-02-02T04:54:41.095204Z',
      experimentId: 6,
      resources: {
        'checkpoint_file': 3,
        'workload_sequencer.pkl': 88,
      },
      state: 'COMPLETED',
      totalBatches: 100,
      trialId: 6,
      uuid: 'f2684332-98e1-4a78-a1f7-c8107f15db2a',
    },
  };
  const [xAxis, setXAxis] = useState<XAxisDomain>(XAxisDomain.Batches);
  const createChartGrid = useChartGrid();
  return (
    <ComponentSection id="Charts" title="Charts">
      <AntDCard>
        <p>
          Line Charts (<code>{'<LineChart>'}</code>) are a universal component to create charts for
          learning curve, metrics, cluster history, etc. We currently use the uPlot library.
        </p>
      </AntDCard>
      <AntDCard title="Label options">
        <p>A chart with two metrics, a title, a legend, an x-axis label, a y-axis label.</p>
        <div>
          <Button onClick={randomizeLineData}>Randomize line data</Button>
          <Button onClick={streamLineData}>Stream line data</Button>
        </div>
        <LineChart
          handleError={handleError}
          height={250}
          series={[line1, line2]}
          showLegend={true}
          title="Sample"
        />
      </AntDCard>
      <AntDCard title="Focus series">
        <p>Highlight a specific metric in the chart.</p>
        <div>
          <Button onClick={randomizeLineData}>Randomize line data</Button>
          <Button onClick={streamLineData}>Stream line data</Button>
        </div>
        <LineChart
          focusedSeries={1}
          handleError={handleError}
          height={250}
          series={[line1, line2]}
          title="Sample"
        />
      </AntDCard>
      <AntDCard title="Series with all x=0">
        <p>When all points have x=0, the x-axis bounds should go from 0 to 1.</p>
        <LineChart
          handleError={handleError}
          height={250}
          series={[zeroline]}
          title="Series with all x=0"
        />
      </AntDCard>
      <AntDCard title="States without data">
        <strong>Loading</strong>
        <LineChart
          handleError={handleError}
          height={250}
          series={NotLoaded}
          showLegend={true}
          title="Loading state"
        />
        <hr />
        <strong>Empty</strong>
        <LineChart
          handleError={handleError}
          height={250}
          series={[]}
          showLegend={true}
          title="Empty state"
        />
      </AntDCard>
      <AntDCard title="Chart Grid">
        <p>
          A Chart Grid (<code>{'<ChartGrid>'}</code>) can be used to place multiple charts in a
          responsive grid. There is a sync for the plot window, cursor, and selection/zoom of an
          x-axis range. There will be a linear/log scale switch, and if multiple X-axis options are
          provided, an X-axis switch.
        </p>
        {createChartGrid({
          chartsProps: [
            {
              plugins: [
                drawPointsPlugin(checkpointsDict),
                tooltipsPlugin({
                  getXTooltipHeader(xIndex) {
                    const xVal = line1.data[xAxis]?.[xIndex]?.[0];

                    if (xVal === undefined) return '';
                    const checkpoint = checkpointsDict?.[Math.floor(xVal)];
                    if (!checkpoint) return '';
                    return '<div>⬦ Best Checkpoint <em>(click to view details)</em> </div>';
                  },
                  isShownEmptyVal: false,
                  seriesColors: ['#009BDE'],
                }),
              ],
              series: [line1],
              showLegend: true,
              title: 'Sample1',
              xAxis,
              xLabel: xAxis,
            },
            {
              series: [line2, line3],
              showLegend: true,
              title: 'Sample2',
              xAxis,
              xLabel: xAxis,
            },
          ],
          handleError,
          onXAxisChange: setXAxis,
          xAxis: xAxis,
        })}
        <hr />
        <strong>Loading</strong>
        {createChartGrid({
          chartsProps: NotLoaded,
          handleError,
          onXAxisChange: setXAxis,
          xAxis: xAxis,
        })}
        <hr />
        <strong>Empty</strong>
        {createChartGrid({
          chartsProps: [],
          handleError,
          onXAxisChange: setXAxis,
          xAxis: xAxis,
        })}
      </AntDCard>
    </ComponentSection>
  );
};

const CheckboxesSection: React.FC = () => {
  return (
    <ComponentSection id="Checkboxes" title="Checkboxes">
      <AntDCard>
        <p>
          Checkboxes (<code>{'<Checkbox>'}</code>) give people a way to select one or more items
          from a group, or switch between two mutually exclusive options (checked or unchecked, on
          or off).
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Use a single check box when there&apos;s only one selection to make or choice to
            confirm. Selecting a blank check box selects it. Selecting it again clears the check
            box.
          </li>
          <li>
            Use multiple check boxes when one or more options can be selected from a group. Unlike
            radio buttons, selecting one check box will not clear another check box.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Separate two groups of check boxes with headings rather than positioning them one after
            the other.
          </li>
          <li>Use sentence-style capitalization—only capitalize the first word.</li>
          <li>
            Don&apos;t use end punctuation (unless the check box label absolutely requires multiple
            sentences).
          </li>
          <li>Use a sentence fragment for the label, rather than a full sentence.</li>
          <li>
            Make it easy for people to understand what will happen if they select or clear a check
            box.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Basic checkboxes</strong>
        <Checkbox>This is a basic checkbox.</Checkbox>
        <strong>Variations</strong>
        <Checkbox checked>Checked checkbox</Checkbox>
        <Checkbox checked={false}>Unchecked checkbox</Checkbox>
        <Checkbox checked disabled>
          Disabled checked checkbox
        </Checkbox>
        <p>Mandatory checkbox - not implemented.</p>
        <p>Mandatory checkbox with info sign - not implemented.</p>
        <Checkbox indeterminate>Indeterminate checkbox</Checkbox>
      </AntDCard>
    </ComponentSection>
  );
};

const ClipboardButtonSection: React.FC = () => {
  const defaultContent = 'This is the content to copy to clipboard.';
  const [content, setContent] = useState(defaultContent);
  const getContent = useCallback(() => content, [content]);
  return (
    <ComponentSection id="ClipboardButton" title="ClipboardButton">
      <AntDCard>
        <p>
          ClipboardButton (<code>{'<ClipboardButton>'}</code> provides a special button for the
          purpose of copying some text into the browser clipboard.
          <br />
          <b>Note:</b> This capability is only available on `https` and `localhost` hosts. `http`
          protocol is purposefully blocked for&nbsp;
          <a href="https://developer.mozilla.org/en-US/docs/Web/API/Clipboard">security reasons</a>.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <Label>Copy Content</Label>
        <Input value={content} onChange={(s) => setContent(String(s.target.value))} />
        <hr />
        <strong>Default Clipboard Button</strong>
        <ClipboardButton getContent={getContent} />
        <strong>Disabled Clipboard Button</strong>
        <ClipboardButton disabled getContent={getContent} />
        <strong>Custom Copied Message Clipboard Button</strong>
        <ClipboardButton copiedMessage="Yay it's copied!" getContent={getContent} />
      </AntDCard>
    </ComponentSection>
  );
};

const DropdownSection: React.FC = () => {
  const menu: MenuItem[] = [
    { key: 'start', label: 'Start' },
    { key: 'stop', label: 'Stop' },
  ];
  const menuWithDivider: MenuItem[] = [
    ...menu,
    { type: 'divider' },
    { key: 'archive', label: 'Archive' },
  ];
  const menuWithDanger: MenuItem[] = [...menu, { danger: true, key: 'delete', label: 'Delete' }];
  const menuWithDisabled: MenuItem[] = [
    ...menu,
    { disabled: true, key: 'delete', label: 'Delete' },
  ];

  return (
    <ComponentSection id="Dropdown" title="Dropdown">
      <AntDCard>
        <p>
          A (<code>{'<Dropdown>'}</code>) is used to display a component when triggered by a child
          element (usually a button). This component can be a menu (a list of actions/options
          defined via the <code>{'menu'}</code> prop), or can be any arbitrary component, defined
          via the <code>{'content'}</code> prop, with default styling applied.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Dropdown variations</strong>
        <Space>
          <Dropdown menu={menu}>
            <Button>Dropdown with menu</Button>
          </Dropdown>
          <Space>
            <Dropdown content={<Input />}>
              <Button>Dropdown with component content</Button>
            </Dropdown>
          </Space>
          <Dropdown disabled menu={menu}>
            <Button>Disabled Dropdown menu</Button>
          </Dropdown>
        </Space>
        <strong>Dropdown menu variations</strong>
        <Space>
          <Dropdown menu={menuWithDivider}>
            <Button>Dropdown menu with a Divider</Button>
          </Dropdown>
          <Dropdown menu={menuWithDanger}>
            <Button>Dropdown menu with Dangerous Option</Button>
          </Dropdown>
          <Dropdown menu={menuWithDisabled}>
            <Button>Dropdown menu with Disabled Option</Button>
          </Dropdown>
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const UncontrolledCodeEditor = () => {
  const [path, setPath] = useState<string>('one.yaml');
  const file = useMemo(() => {
    if (!path) {
      return NotLoaded;
    }
    return (
      {
        'one.yaml': Loaded(
          'hyperparameters:\n  learning_rate: 1.0\n  global_batch_size: 512\n  n_filters1: 32\n  n_filters2: 64\n  dropout1: 0.25\n  dropout2: 0.5',
        ),
        'two.yaml': Loaded('searcher:\n  name: single\n  metric: validation_loss\n'),
      }[path] || NotLoaded
    );
  }, [path]);
  return (
    <CodeEditor
      file={file}
      files={[
        {
          isLeaf: true,
          key: 'one.yaml',
          title: 'one.yaml',
        },
        {
          isLeaf: true,
          key: 'two.yaml',
          title: 'two.yaml',
        },
        {
          isLeaf: true,
          key: 'unloaded.yaml',
          title: 'unloaded.yaml',
        },
      ]}
      readonly={true}
      selectedFilePath={path}
      onError={handleError}
      onSelectFile={setPath}
    />
  );
};
const CodeEditorSection: React.FC = () => {
  return (
    <ComponentSection id="CodeEditor" title="CodeEditor">
      <AntDCard>
        <p>
          The Code Editor (<code>{'<CodeEditor>'}</code>) shows Python and YAML files with syntax
          highlighting. If multiple files are sent, the component shows a file tree browser.
        </p>
        <ul>
          <li>Use the readonly attribute to make code viewable but not editable.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Editable Python file</strong>
        <CodeEditor
          file={Loaded('import math\nprint(math.pi)\n\n')}
          files={[
            {
              key: 'test.py',
              title: 'test.py',
            },
          ]}
          onError={handleError}
        />
        <strong>Read-only YAML file</strong>
        <CodeEditor
          file={Loaded(
            'name: Unicode Test 日本😃\ndata:\n  url: https://example.tar.gz\nhyperparameters:\n  learning_rate: 1.0\n  global_batch_size: 64\n  n_filters1: 32\n  n_filters2: 64\n  dropout1: 0.25\n  dropout2: 0.5\nsearcher:\n  name: single\n  metric: validation_loss\n  max_length:\n      batches: 937 #60,000 training images with batch size 64\n  smaller_is_better: true\nentrypoint: model_def:MNistTrial\nresources:\n  slots_per_trial: 2',
          )}
          files={[
            {
              key: 'test1.yaml',
              title: 'test1.yaml',
            },
          ]}
          readonly={true}
          onError={handleError}
        />
        <strong>Multiple files, one not finished loading.</strong>
        <UncontrolledCodeEditor />
      </AntDCard>
    </ComponentSection>
  );
};

const InlineFormSection: React.FC = () => {
  const [inputWithValidatorValue, setInputWithValidatorValue] = useState('');
  const [searchInput, setSearchInput] = useState('');
  const [numberInput, setNumberInput] = useState(1234);
  const [textAreaValue, setTextAreaValue] = useState(loremIpsumSentence);
  const [passwordInputValue, setPasswordInputValue] = useState('123456789');
  const [selectValue, setSelectValue] = useState('off');

  const inputWithValidatorCallback = useCallback((newValue: string) => {
    setInputWithValidatorValue(newValue);
  }, []);
  const numberInputCallback = useCallback((newValue: number) => {
    setNumberInput(newValue);
  }, []);
  const searchCallback = useCallback((newValue: string) => {
    setSearchInput(newValue);
  }, []);
  const textAreaCallback = useCallback((newValue: string) => {
    setTextAreaValue(newValue);
  }, []);
  const passwordInputCallback = useCallback((newValue: string) => {
    setPasswordInputValue(newValue);
  }, []);
  const selectCallback = useCallback((newValue: string) => {
    setSelectValue(newValue === '1' ? 'off' : 'on');
  }, []);

  return (
    <ComponentSection id="InlineForm" title="InlineForm">
      <AntDCard>
        <p>
          The <code>{'<InlineForm>'}</code> allows people to have a simple form with just one input
          to interact with.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <p>
          If using the <code>{'Input.Password'}</code> component, is important to pass the{' '}
          <code>{'isPassword'}</code> prop.
        </p>
        <br />
        <h5>Controlled</h5>
        <div style={{ maxWidth: '700px' }}>
          <InlineForm<string>
            initialValue={''}
            label="Input with validator"
            rules={[{ message: 'Please input something here!', required: true }]}
            value={inputWithValidatorValue}
            onSubmit={inputWithValidatorCallback}>
            <Input maxLength={32} />
          </InlineForm>
          <hr />
          <InlineForm<string>
            initialValue={textAreaValue}
            label="Text Area"
            value={textAreaValue}
            onSubmit={textAreaCallback}>
            <Input.TextArea />
          </InlineForm>
          <hr />
          <InlineForm<string>
            initialValue={''}
            isPassword
            label="Password"
            value={passwordInputValue}
            onSubmit={passwordInputCallback}>
            <Input.Password />
          </InlineForm>
          <hr />
          <InlineForm<string>
            initialValue={selectValue}
            label="Select"
            value={selectValue}
            onSubmit={selectCallback}>
            <Select defaultValue={1} searchable={false}>
              {[
                { label: 'off', value: 1 },
                { label: 'on', value: 2 },
              ].map((opt) => (
                <Option key={opt.value as React.Key} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </InlineForm>
          <hr />
          <InlineForm<number>
            initialValue={numberInput}
            label="Input Number"
            value={numberInput}
            onSubmit={numberInputCallback}>
            <InputNumber />
          </InlineForm>
          <hr />
          <InlineForm<string>
            initialValue={searchInput}
            label="Input Search"
            value={searchInput}
            onSubmit={searchCallback}>
            <InputSearch allowClear enterButton placeholder="Input Search" />
          </InlineForm>
        </div>
        <h5>Uncontrolled</h5>
        <div style={{ maxWidth: '700px' }}>
          <InlineForm<string>
            initialValue={'initial value'}
            label="Input with validator"
            rules={[{ message: 'Please input something here!', required: true }]}>
            <Input />
          </InlineForm>
          <hr />
          <InlineForm<string> initialValue={textAreaValue} label="Text Area">
            <Input.TextArea />
          </InlineForm>
          <hr />
          <InlineForm<string> initialValue={''} isPassword label="Password">
            <Input.Password />
          </InlineForm>
          <hr />
          <InlineForm<string> initialValue={selectValue} label="Select">
            <Select defaultValue={1} searchable={false}>
              {[
                { label: 'off', value: 1 },
                { label: 'on', value: 2 },
              ].map((opt) => (
                <Option key={opt.value as React.Key} value={opt.value}>
                  {opt.label}
                </Option>
              ))}
            </Select>
          </InlineForm>
          <hr />
          <InlineForm<number> initialValue={1234} label="Input Number">
            <InputNumber />
          </InlineForm>
          <hr />
          <InlineForm<string> initialValue={''} label="Input Search">
            <InputSearch allowClear enterButton placeholder="Input Search" />
          </InlineForm>
        </div>
      </AntDCard>
    </ComponentSection>
  );
};

const InputSearchSection: React.FC = () => {
  return (
    <ComponentSection id="InputSearch" title="InputSearch">
      <AntDCard>
        <p>
          A search box (<code>{'<InputSearch>'}</code>) provides an input field for searching
          content within a site or app to find specific items.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Don&apos;t build a custom search control based on the default text box or any other
            control.
          </li>
          <li>
            Use a search box without a parent container when it&apos;s not restricted to a certain
            width to accommodate other content. This search box will span the entire width of the
            space it&apos;s in.
          </li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Use placeholder text in the search box to describe what people can search for. For
            example, &quot;Search&quot;, &quot;Search files&quot;, or &quot;Search contacts
            list&quot;.
          </li>
          <li>
            Although search entry points tend to be similarly visualized, they can provide access to
            results that range from broad to narrow. By effectively communicating the scope of a
            search, you can ensure that people&apos;s expectations are met by the capabilities of
            the search you&apos;re performing, which will reduce the possibility of frustration. The
            search entry point should be placed near the content being searched.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default Searchbox</strong>
        <InputSearch placeholder="input search text" />
        <strong>Variations</strong>
        <InputSearch allowClear enterButton value="Active search box" />
        <InputSearch disabled placeholder="disabled search box" />
        <hr />
        <strong>In-table Searchbox</strong>
        <p>Not implemented</p>
        <hr />
        <strong>Search box with scopes</strong>
        <p>Not implemented</p>
      </AntDCard>
    </ComponentSection>
  );
};

const InputShortcutSection: React.FC = () => {
  const [value, setValue] = useState<KeyboardShortcut>();
  const onChange = (k: KeyboardShortcut | undefined) => {
    setValue(k);
  };
  return (
    <ComponentSection id="InputShortcut" title="InputShortcut">
      <AntDCard>
        <p>
          An input box (<code>{'<InputShortcut>'}</code>) for keyboard shortcuts.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default Input for Shortcut</strong>
        <InputShortcut />
        <strong>Controlled Input for Shortcut</strong>
        <InputShortcut value={value} onChange={onChange} />
      </AntDCard>
    </ComponentSection>
  );
};

const InputNumberSection: React.FC = () => {
  return (
    <ComponentSection id="InputNumber" title="InputNumber">
      <AntDCard>
        <p>
          A spin button (<code>{'<InputNumber>'}</code>) allows someone to incrementally adjust a
          value in small steps. It&apos;s mainly used for numeric values, but other values are
          supported too.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            Place labels to the left of the spin button control. For example, &quot;Length of ruler
            (cm)&quot;.
          </li>
          <li>Spin button width should adjust to fit the number values.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>Use a spin button when you need to incrementally change a value.</li>
          <li>Use a spin button when values are tied to a unit of measure.</li>
          <li>Don&apos;t use a spin button for binary settings.</li>
          <li>Don&apos;t use a spin button for a range of three values or less.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default InputNumber</strong>
        <InputNumber />
        <strong>Disabled InputNumber</strong>
        <InputNumber disabled />
        <hr />
        <span>
          Also see <a href={`#${ComponentTitles.Form}`}>Form</a> for form-specific variations
        </span>
      </AntDCard>
    </ComponentSection>
  );
};

const InputSection: React.FC = () => {
  return (
    <ComponentSection id="Input" title="Input">
      <AntDCard>
        <p>
          Text fields (<code>{'<Input>'}</code>) give people a way to enter and edit text.
          They&apos;re used in forms, modal dialogs, tables, and other surfaces where text input is
          required.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>Use a multiline text field when long entries are expected.</li>
          <li>
            Don&apos;t place a text field in the middle of a sentence, because the sentence
            structure might not make sense in all languages. For example, &quot;Remind me in
            [textfield] weeks&quot; should instead read, &quot;Remind me in this many weeks:
            [textfield]&quot;.
          </li>
          <li>Format the text field for the expected entry.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>
          Input <code>{'<Input>'}</code>
        </strong>
        <strong>Default Input</strong>
        <Input />
        <strong>Disabled Input</strong>
        <Input disabled />
        <hr />
        <strong>
          TextArea <code>{'<Input.TextArea>'}</code>
        </strong>
        <strong>Default TextArea</strong>
        <Input.TextArea />
        <strong>Disabled TextArea</strong>
        <Input.TextArea disabled />
        <hr />
        <strong>
          Password <code>{'<Input.Password>'}</code>
        </strong>
        <strong>Default Password</strong>
        <Input.Password />
        <strong>Disabled Password</strong>
        <Input.Password disabled />
        <hr />
        <span>
          Also see <a href={`#${ComponentTitles.Form}`}>Form</a> for form-specific variations
        </span>
      </AntDCard>
    </ComponentSection>
  );
};

const ListsSection: React.FC = () => {
  const mockColumns = [
    {
      dataIndex: 'id',
      sorter: true,
      title: 'ID',
    },
    {
      dataIndex: 'name',
      sorter: true,
      title: 'Name',
    },
  ];

  const mockRows = [
    {
      id: 'Row id',
      name: 'Row name',
    },
  ];

  return (
    <ComponentSection id="Lists" title="Lists (tables)">
      <AntDCard>
        <p>
          A list (<code>{'<ResponsiveTable>'}</code>) is a robust way to display an information-rich
          collection of items, and allow people to sort, group, and filter the content. Use a
          details list when information density is critical.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Layout</strong>
        <ul>
          <li>
            List items are composed of selection, icon, and name columns at minimum. You can include
            other columns, such as date modified, or any other metadata field associated with the
            collection.
          </li>
          <li>
            Avoid using file type icon overlays to denote status of a file as it can make the entire
            icon unclear.
          </li>
          <li>
            If there are multiple lines of text in a column, consider the variable row height
            variant.
          </li>
          <li>Give columns ample default width to display information.</li>
        </ul>
        <strong>Content</strong>
        <ul>
          <li>
            Use sentence-style capitalization for column headers—only capitalize the first word.
          </li>
        </ul>
        <strong>Accessibility</strong>
        <ul>
          <li>
            When creating a DetailsList where one column is clearly the primary label for the row,
            it&apos;s best to use isRowHeader on that column to create a better screen reader
            experience navigating the table. For selectable DetailsLists, specifying a row header
            also gives the checkboxes a better accessible label.
          </li>
        </ul>
        <strong>Keyboard hotkeys</strong>
        <ul>
          <li>
            DetailsList supports different selection modes with keyboard behavior differing based on
            the current selection mode.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default list</strong>
        <ResponsiveTable columns={mockColumns} dataSource={mockRows} rowKey="id" />
      </AntDCard>
    </ComponentSection>
  );
};

const DatePickerSection: React.FC = () => {
  return (
    <ComponentSection id="DatePicker" title="DatePicker">
      <AntDCard>
        <p>
          <code>DatePicker</code> is a form element for the user to select a specific time, date, or
          month from a calendar UI. When using <code>onChange</code>, the returned value is a{' '}
          <code>Dayjs</code> object. The component accepts a subset of the props for the{' '}
          <code>Antd.DatePicker</code>, with the <code>style</code> prop replaced by our usage (
          <code>width</code>).
        </p>
        <p>
          The <code>picker</code> prop can be set to select a month. Alternatively the{' '}
          <code>showTime</code> prop adds precision to the second.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        DatePickers with labels:
        <strong>Date-time picker</strong>
        <DatePicker label="Choose a date and time" showTime onChange={noOp} />
        <strong>Clearable day picker</strong>
        <DatePicker label="Choose a date" onChange={noOp} />
        <hr />
        <strong>Un-clearable month picker, without a label</strong>
        <DatePicker allowClear={false} picker="month" onChange={noOp} />
      </AntDCard>
    </ComponentSection>
  );
};

const BreadcrumbsSection: React.FC = () => {
  const menuItems: MenuItem[] = [
    { key: 'Action 1', label: 'Action 1' },
    { key: 'Action 2', label: 'Action 2' },
  ];

  return (
    <ComponentSection id="Breadcrumbs" title="Breadcrumbs">
      <AntDCard>
        <p>
          <code>{'<Breadcrumb>'}</code>s should be used as a navigational aid in your app or site.
          They indicate the current page&apos;s location within a hierarchy and help the user
          understand where they are in relation to the rest of that hierarchy. They also afford
          one-click access to higher levels of that hierarchy.
        </p>
        <p>
          Breadcrumbs are typically placed, in horizontal form, under the masthead or navigation of
          an experience, above the primary content area.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Accessibility</strong>
        <ul>
          <li>By default, Breadcrumb uses arrow keys to cycle through each item. </li>
          <li>
            Place Breadcrumbs at the top of a page, above a list of items, or above the main content
            of a page.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Breadcrumb</strong>
        <Breadcrumb>
          <Breadcrumb.Item>Level 0</Breadcrumb.Item>
          <Breadcrumb.Item>Level 1</Breadcrumb.Item>
          <Breadcrumb.Item>Level 2</Breadcrumb.Item>
        </Breadcrumb>
        <strong>Breadcrumb with actions</strong>
        <Breadcrumb menuItems={menuItems}>
          <Breadcrumb.Item>Level 0</Breadcrumb.Item>
          <Breadcrumb.Item>Level 1</Breadcrumb.Item>
        </Breadcrumb>
      </AntDCard>
    </ComponentSection>
  );
};

const useNoteDemo = (): ((props?: Omit<NotesProps, 'multiple'>) => JSX.Element) => {
  const [note, setNote] = useState<Note>({ contents: '', name: 'Untitled' });
  const onSave = async (n: Note) => await setNote(n);
  return (props) => <Notes onError={handleError} {...props} notes={note} onSave={onSave} />;
};

const useNotesDemo = (): ((props?: NotesProps) => JSX.Element) => {
  const [notes, setNotes] = useState<Note[]>([]);
  const onDelete = (p: number) => setNotes((n) => n.filter((_, idx) => idx !== p));
  const onNewPage = () => setNotes((n) => [...n, { contents: '', name: 'Untitled' }]);
  const onSave = async (n: Note[]) => await setNotes(n);
  return (props) => (
    <Notes
      {...props}
      multiple
      notes={notes}
      onDelete={onDelete}
      onError={handleError}
      onNewPage={onNewPage}
      onSave={onSave}
    />
  );
};

const NotesSection: React.FC = () => {
  return (
    <ComponentSection id="Notes" title="Notes">
      <AntDCard>
        <p>
          A <code>{'<Notes>'}</code> is used for taking notes. It can be single page note or multi
          pages notes. Each page of note consists of a title and a sheet of note.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Single page note</strong>
        {useNoteDemo()()}
        <hr />
        <strong>Multi pages notes</strong>
        {useNotesDemo()()}
      </AntDCard>
    </ComponentSection>
  );
};

const AvatarSection: React.FC = () => {
  return (
    <ComponentSection id="Avatar" title="Avatar">
      <AntDCard>
        <p>
          An avatar (<code>{'<Avatar>'}</code>) is a compact information display. The information is
          abbreviated with an option to hover for an unabbreviated view.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <Avatar displayName="Test User" />
      </AntDCard>
    </ComponentSection>
  );
};

const NameplateSection: React.FC = () => {
  const testUser = { displayName: 'Test User', id: 1, username: 'testUser123' } as const;

  return (
    <ComponentSection id="Nameplate" title="Nameplate">
      <AntDCard>
        <p>
          A (<code>{'<Nameplate>'}</code>) displays an icon, a name, and an optional alias. The icon
          is displayed on the left, and the text fields are displayed on the right. If an alias is
          provided, it is displayed above the name in larger font. A &apos;compact&apos; option
          reduces the size of the name for use in a smaller form or modal.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <li>With name and alias</li>
        <Nameplate
          alias={testUser.displayName}
          icon={<Avatar displayName={testUser.displayName} />}
          name={testUser.username}
        />
        <li>Compact format</li>
        <Nameplate
          alias={testUser.displayName}
          compact
          icon={<Avatar displayName={testUser.displayName} />}
          name={testUser.username}
        />
        <li>No alias</li>
        <Nameplate icon={<Icon name="group" title="Group" />} name="testGroup123" />
        <li>Compact, no alias</li>
        <Nameplate compact icon={<Icon name="group" title="Group" />} name="testGroup123" />
      </AntDCard>
    </ComponentSection>
  );
};

const PivotSection: React.FC = () => {
  return (
    <ComponentSection id="Pivot" title="Pivot">
      <AntDCard>
        <p>
          The Pivot control (<code>{'<Tabs>'}</code>) and related tabs pattern are used for
          navigating frequently accessed, distinct content categories. Pivots allow for navigation
          between two or more content views and relies on text headers to articulate the different
          sections of content.
        </p>
        <p>Tapping on a pivot item header navigates to that header&apos;s section content.</p>
        <p>
          Tabs are a visual variant of Pivot that use a combination of icons and text or just icons
          to articulate section content.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            Be concise on the navigation labels, ideally one or two words rather than a phrase.
          </li>
          <li>
            Use on content-heavy pages that require a significant amount of scrolling to access the
            various sections.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Primary Pivot</strong>
        <Space>
          <Pivot
            items={[
              { children: 'Overview', key: 'Overview', label: 'Overview' },
              { children: 'Hyperparameters', key: 'hyperparameters', label: 'Hyperparameters' },
              { children: 'Checkpoints', key: 'checkpoints', label: 'Checkpoints' },
              { children: 'Code', key: 'code', label: 'Code' },
              { children: 'Notes', key: 'notes', label: 'Notes' },
              { children: 'Profiler', key: 'profiler', label: 'Profiler' },
              { children: 'Logs', key: 'logs', label: 'Logs' },
            ]}
          />
        </Space>
        <hr />
        <strong>Secondary Pivot</strong>
        <Space>
          <Pivot
            items={[
              { children: 'Overview', key: 'Overview', label: 'Overview' },
              { children: 'Hyperparameters', key: 'hyperparameters', label: 'Hyperparameters' },
              { children: 'Checkpoints', key: 'checkpoints', label: 'Checkpoints' },
              { children: 'Code', key: 'code', label: 'Code' },
              { children: 'Notes', key: 'notes', label: 'Notes' },
              { children: 'Profiler', key: 'profiler', label: 'Profiler' },
              { children: 'Logs', key: 'logs', label: 'Logs' },
            ]}
            type="secondary"
          />
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const PaginationSection: React.FC = () => {
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [currentPageSize, setCurrentPageSize] = useState<number>(1);

  return (
    <ComponentSection id="Pagination" title="Pagination">
      <AntDCard>
        <p>
          <code>{'<Pagination>'}</code> is the process of splitting the contents of a website, or
          section of contents from a website, into discrete pages. This user interface design
          pattern is used so users are not overwhelmed by a mass of data on one page. Page breaks
          are automatically set.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>Use ordinal numerals or letters of the alphabet.</li>
          <li>
            Indentify the current page in addition to the pages in immediate context/surrounding.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Pagination default</strong>
        <Pagination
          current={currentPage}
          pageSize={currentPageSize}
          total={500}
          onChange={(page: number, pageSize: number) => {
            setCurrentPage(page);
            setCurrentPageSize(pageSize);
          }}
        />
        <strong>Considerations</strong>
        <ul>
          <li>
            Always give the user the option to navigate to the first & last page -- this helps with
            sorts.
          </li>
          <li>
            Provide the user with 2 pages before/after when navigating &apos;island&apos;
            page-counts.
          </li>
          <li>Provide the user with the first 4 or last 4 pages of the page-range.</li>
          <li>
            Ensure the FocusTrap is set to the whole pagination component so that user doesn&apos;t
            tabs in/out accidentally.
          </li>
        </ul>
      </AntDCard>
    </ComponentSection>
  );
};

const CardsSection: React.FC = () => {
  return (
    <ComponentSection id="Cards" title="Cards">
      <AntDCard>
        <p>
          A Card (<code>{'<Card>'}</code>) contains additional metadata or actions. This offers
          people a richer view into a file than the typical grid view.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>Incorporate metadata that is relevant and useful in this particular view.</li>
          <li>
            Don&apos;t use a document card in views where someone is likely to be performing bulk
            operations in files, or when the list may get very long. Specifically, if you&apos;re
            showing all the items inside an actual folder, a card may be overkill because the
            majority of the items in the folder may not have interesting metadata.
          </li>
          <li>
            Don&apos;t use a document card if space is at a premium or you can&apos;t show relevant
            and timely commands or metadata. Cards are useful because they can expose on-item
            interactions like “Share” buttons or view counts.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Card default</strong>
        <Card />
        <strong>Card group default</strong>
        <p>
          A card group (<code>{'<Card.Group>'}</code>) can be used to display a list or grid of
          cards.
        </p>
        <Card.Group>
          <Card />
          <Card />
        </Card.Group>
        <strong>Considerations</strong>
        <ul>
          <li>Ensure links are tab-able.</li>
          <li>Ensure data is relevant and if not, remove it.</li>
          <li>We need to revisit the density of each of the cards and content.</li>
          <li>
            Implement quick actions inside of the card as to prevent the user from providing
            additional clicks.
          </li>
        </ul>
        <strong>Card variations</strong>
        <p>Small cards (default)</p>
        <Card.Group>
          <Card actionMenu={[{ key: 'test', label: 'Test' }]}>Card with actions</Card>
          <Card actionMenu={[{ key: 'test', label: 'Test' }]} disabled>
            Disabled card
          </Card>
          <Card onClick={noOp}>Clickable card</Card>
        </Card.Group>
        <p>Medium cards</p>
        <Card.Group size="medium">
          <Card actionMenu={[{ key: 'test', label: 'Test' }]} size="medium">
            Card with actions
          </Card>
          <Card actionMenu={[{ key: 'test', label: 'Test' }]} disabled size="medium">
            Disabled card
          </Card>
          <Card size="medium" onClick={noOp}>
            Clickable card
          </Card>
        </Card.Group>
        <strong>Card group variations</strong>
        <p>Wrapping group (default)</p>
        <Card.Group size="medium">
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
        </Card.Group>
        <p>Non-wrapping group</p>
        <Card.Group size="medium" wrap={false}>
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
          <Card size="medium" />
        </Card.Group>
      </AntDCard>
    </ComponentSection>
  );
};

const LogViewerSection: React.FC = () => {
  const sampleLogs = [
    {
      id: 1,
      level: V1LogLevel.INFO,
      message: 'Determined master 0.19.7-dev0 (built with go1.18.7)',
      timestamp: '2022-06-02T21:48:07.456381-06:00',
    },
    {
      id: 2,
      level: V1LogLevel.INFO,
      message:
        'connecting to database determined-master-database-tytmqsutj5d1.cluster-csrkoc1nkoog.us-west-2.rds.amazonaws.com:5432',
      timestamp: '2022-07-02T21:48:07.456381-06:00',
    },
    {
      id: 3,
      level: V1LogLevel.INFO,
      message:
        'running DB migrations from file:///usr/share/determined/master/static/migrations; this might take a while...',
      timestamp: '2022-08-02T21:48:07.456381-06:00',
    },
    {
      id: 4,
      level: V1LogLevel.INFO,
      message: 'no migrations to apply; version: 20221026124235',
      timestamp: '2022-09-02T21:48:07.456381-06:00',
    },
    {
      id: 5,
      level: V1LogLevel.ERROR,
      message:
        'failed to aggregate resource allocation: failed to add aggregate allocation: ERROR: range lower bound must be less than or equal to range upper bound (SQLSTATE 22000)  actor-local-addr="allocation-aggregator" actor-system="master" go-type="allocationAggregator"',
      timestamp: '2022-10-02T21:48:07.456381-06:00',
    },
    {
      id: 6,
      level: V1LogLevel.WARNING,
      message:
        'received update on unknown agent  actor-local-addr="aux-pool" actor-system="master" agent-id="i-018fadb36ddbfe97a" go-type="ResourcePool" resource-pool="aux-pool"',
      timestamp: '2022-11-02T21:48:07.456381-06:00',
    },
  ];
  return (
    <ComponentSection id="LogViewer" title="LogViewer">
      <AntDCard>
        <p>
          A Logview (<code>{'<LogViewer>'}</code>) prints events that have been configured to be
          triggered and return them to the user in a running stream.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            Prioritize accessibility and readability of the log entry as details can always be
            generated afterwards.
          </li>
          <li>
            Prioritize IntelliSense type of readability improvements as it helps scannability of the
            text.
          </li>
          <li>Provide the user with ways of searching & filtering down logs.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>LogViewer default</strong>
        <div style={{ height: '300px' }}>
          <LogViewer
            decoder={mapV1LogsResponse}
            initialLogs={sampleLogs}
            serverAddress={serverAddress}
            sortKey="id"
            onError={handleError}
          />
        </div>
        <strong>Considerations</strong>
        <ul>
          <li>
            Ensure that we don&apos;t overload the users with information --&gt; we need to know
            what events we&apos;re listening to.
          </li>
          <li>Ensure the capability of searching/filtering log entries.</li>
        </ul>
      </AntDCard>
    </ComponentSection>
  );
};

const FormSection: React.FC = () => {
  return (
    <ComponentSection id="Form" title="Form">
      <AntDCard>
        <p>
          <code>{'<Form>'}</code> and <code>{'<Form.Item>'}</code> components are used for
          submitting user input. When these components wrap a user input field (such as{' '}
          <code>{'<Input>'}</code> or <code>{'<Select>'}</code>), they can show a standard label,
          indicate that the field is required, apply input validation, or display an input
          validation error.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <Form>
          <strong>
            Form-specific <a href={ComponentTitles.Input}>Input</a> variations
          </strong>
          <br />
          <Form.Item label="Required input" name="required_input" required>
            <Input />
          </Form.Item>
          <Form.Item
            label="Invalid input"
            name="invalid_input"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific <a href={ComponentTitles.Input}>TextArea</a> variations
          </strong>
          <br />
          <Form.Item label="Required TextArea" name="required_textarea" required>
            <Input.TextArea />
          </Form.Item>
          <Form.Item
            label="Invalid TextArea"
            name="invalid_textarea"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input.TextArea />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific <a href={ComponentTitles.Input}>Password</a> variations
          </strong>
          <br />
          <Form.Item label="Required Password" name="required_label" required>
            <Input.Password />
          </Form.Item>
          <Form.Item
            label="Invalid Password"
            name="invalid_password"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input.Password />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific <a href={ComponentTitles.Input}>InputNumber</a> variations
          </strong>
          <Form.Item label="Required InputNumber" name="number" required>
            <InputNumber />
          </Form.Item>
          <Form.Item
            label="Invalid InputNumber"
            validateMessage="Input validation error"
            validateStatus="error">
            <InputNumber />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific <a href={ComponentTitles.Select}>Select</a> variations
          </strong>
          <Form.Item initialValue={1} label="Required dropdown" name="required_dropdown" required>
            <Select
              options={[
                { label: 'Option 1', value: 1 },
                { label: 'Option 2', value: 2 },
                { label: 'Option 3', value: 3 },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="Invalid dropdown"
            validateMessage="Input validation error"
            validateStatus="error">
            <Select />
          </Form.Item>
        </Form>
      </AntDCard>
    </ComponentSection>
  );
};

const TagsSection: React.FC = () => {
  const tags: string[] = ['working', 'TODO'];
  const moreTags: string[] = ['working', 'TODO', 'tag1', 'tag2', 'tag3', 'tag4', 'tag5'];
  return (
    <ComponentSection id="Tags" title="Tags">
      <AntDCard>
        <p>
          The editable tags list (<code>{'<Tags>'}</code>) supports &quot;add&quot;,
          &quot;edit&quot; and &quot;remove&quot; actions on individual tags.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content</strong>
        <ul>
          <li>Don&apos;t use tags of the same content within one list.</li>
          <li>Tags are ordered alphabetically.</li>
          <li>Individual tags cannot be empty.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Tags default</strong>
        <Space>{useTags([...tags])()}</Space>
        <strong>Tags ghost</strong>
        <Space>{useTags([...tags])({ ghost: true })}</Space>
        <strong>Tags disabled</strong>
        <Space>{useTags([...tags])({ disabled: true })}</Space>
        <strong>Tags compact</strong>
        <Space>{useTags([...moreTags])({ compact: true })}</Space>
        <strong>Tags with long text</strong>
        <Space>
          {useTags([
            'very very very long text, very very very long text, very very very long text, very very very long text.',
          ])()}
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const TypographySection: React.FC = () => {
  return (
    <ComponentSection id="Typography" title="Typography">
      <AntDCard>
        <p>
          The (<code>{'<Header>'}</code>) is a reusable header element.
        </p>
        <p>
          The (<code>{'<Paragraph>'}</code>) is a reusable simple paragraph element.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content</strong>
        <ul>
          <li>
            For Headers, <code>{'<h1>'}</code> is the default.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Typography - Header</strong>
        <Space>
          <Header>Header</Header>
        </Space>
        <strong>Typography - paragraph</strong>
        <Space>
          <Paragraph>this is a paragraph!</Paragraph>
        </Space>
      </AntDCard>
      <AntDCard title="Font Families">
        <Paragraph>For general UI --theme-font-family</Paragraph>
        <Paragraph font="code">For displaying code --theme-font-family-code</Paragraph>
      </AntDCard>
      <AntDCard title="Font Sizing">
        <div>
          <div style={{ display: 'flex', flexDirection: 'column', marginBottom: '30px' }}>
            <Header>Header</Header>
            <Header size={TypographySize.XL}>
              Model Registry - XL (f.s. 28px, line-height 36px)
            </Header>
            <Header size={TypographySize.L}>
              Model Registry - L (f.s. 24px, line-height 32px)
            </Header>
            <Header size={TypographySize.default}>
              Model Registry - default (f.s. 22px line-height 28px)
            </Header>
            <Header size={TypographySize.S}>Model Registry - s (f.s. 18px line-height 23px)</Header>
            <Header size={TypographySize.XS}>
              Model Registry - xs (f.s. 16px line-height 21px)
            </Header>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', marginBottom: '30px' }}>
            <Header>Multi Line</Header>
            <Paragraph size={TypographySize.XL} type="multi line">
              Lorem ipsum dolor sit amet consectetur, adipisicing elit. Ut suscipit itaque debitis
              amet, eligendi possimus assumenda eos, iusto ea labore, officia aspernatur optio. In
              necessitatibus porro ut vero commodi neque. Lorem ipsum dolor sit amet consectetur
              adipisicing elit. Voluptatibus, omnis quo dolorem magnam dolores necessitatibus iure
              illo incidunt maiores voluptas odit eligendi dignissimos facilis vel veniam id.
              Obcaecati, cum eos. - XL (f.s. 16px line-height 26px)
            </Paragraph>
            <br />
            <Paragraph size={TypographySize.L} type="multi line">
              Lorem ipsum dolor sit amet consectetur, adipisicing elit. Ut suscipit itaque debitis
              amet, eligendi possimus assumenda eos, iusto ea labore, officia aspernatur optio. In
              necessitatibus porro ut vero commodi neque. Lorem ipsum dolor sit amet consectetur
              adipisicing elit. Voluptatibus, omnis quo dolorem magnam dolores necessitatibus iure
              illo incidunt maiores voluptas odit eligendi dignissimos facilis vel veniam id.
              Obcaecati, cum eos. - L (f.s. 14px line-height 22px)
            </Paragraph>
            <br />
            <Paragraph size={TypographySize.default} type="multi line">
              Lorem ipsum dolor sit amet consectetur, adipisicing elit. Ut suscipit itaque debitis
              amet, eligendi possimus assumenda eos, iusto ea labore, officia aspernatur optio. In
              necessitatibus porro ut vero commodi neque. Lorem ipsum dolor sit amet consectetur
              adipisicing elit. Voluptatibus, omnis quo dolorem magnam dolores necessitatibus iure
              illo incidunt maiores voluptas odit eligendi dignissimos facilis vel veniam id.
              Obcaecati, cum eos. - default (f.s. 12px line-height 20px)
            </Paragraph>
            <br />
            <Paragraph size={TypographySize.S} type="multi line">
              Lorem ipsum dolor sit amet consectetur, adipisicing elit. Ut suscipit itaque debitis
              amet, eligendi possimus assumenda eos, iusto ea labore, officia aspernatur optio. In
              necessitatibus porro ut vero commodi neque. Lorem ipsum dolor sit amet consectetur
              adipisicing elit. Voluptatibus, omnis quo dolorem magnam dolores necessitatibus iure
              illo incidunt maiores voluptas odit eligendi dignissimos facilis vel veniam id.
              Obcaecati, cum eos. - s (f.s. 11px line-height 18px)
            </Paragraph>
            <br />
            <Paragraph size={TypographySize.XS} type="multi line">
              Lorem ipsum dolor sit amet consectetur, adipisicing elit. Ut suscipit itaque debitis
              amet, eligendi possimus assumenda eos, iusto ea labore, officia aspernatur optio. In
              necessitatibus porro ut vero commodi neque. Lorem ipsum dolor sit amet consectetur
              adipisicing elit. Voluptatibus, omnis quo dolorem magnam dolores necessitatibus iure
              illo incidunt maiores voluptas odit eligendi dignissimos facilis vel veniam id.
              Obcaecati, cum eos. - xs (f.s. 10px line-height 16px)
            </Paragraph>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', marginBottom: '30px' }}>
            <Header>Single Line</Header>
            <Paragraph size={TypographySize.XL}>
              Model Registry - XL (f.s. 16px line-height 20px)
            </Paragraph>
            <Paragraph size={TypographySize.L}>
              Model Registry - L (f.s. 14px line-height 18px)
            </Paragraph>
            <Paragraph size={TypographySize.default}>
              Model Registry - default (f.s. 12px line-height 16px)
            </Paragraph>
            <Paragraph size={TypographySize.S}>
              Model Registry - s (f.s. 11px line-height 14px)
            </Paragraph>
            <Paragraph size={TypographySize.XS}>
              Model Registry - xs (f.s. 10px line-height 12px)
            </Paragraph>
          </div>
        </div>
      </AntDCard>
    </ComponentSection>
  );
};

const ColorSection: React.FC = () => {
  const themeStatus = Object.values(Status);
  const backgrounds = Object.values(Background);
  const stage = Object.values(Stage);
  const surface = Object.values(Surface);
  const float = Object.values(Float);
  const overlay = Object.values(Overlay);
  const brand = Object.values(Brand);
  const interactive = Object.values(Interactive);

  const renderColorComponent = (colorArray: string[], name: string) => (
    <AntDCard title={`${name} Colors`}>
      <Grid>
        {colorArray.map((cName, idx) => (
          <div
            key={`${idx}-${name.toLowerCase()}`}
            style={{
              marginBottom: '20px',
              width: '250px',
            }}>
            <span>{cName.replace(/(var\(|\))/g, '')}</span>
            <div
              style={{
                backgroundColor: cName,
                border: 'var(--theme-stroke-width) solid var(--theme-surface-border)',
                borderRadius: 'var(--theme-border-radius)',
                height: '40px',
                width: '100%',
              }}
            />
          </div>
        ))}
      </Grid>
    </AntDCard>
  );
  const iterateOverThemes = (themes: Array<string[]>, names: string[]) =>
    themes.map((theme, idx) => renderColorComponent(theme, names[idx]));

  return (
    <ComponentSection id="Color" title="Color">
      <AntDCard>
        <Paragraph>
          We have a variety of colors that are available for use with the components in the UI Kit.
        </Paragraph>
      </AntDCard>
      {iterateOverThemes(
        [themeStatus, backgrounds, stage, surface, float, overlay, brand, interactive],
        ['Status', 'Background', 'Stage', 'Surface', 'Float', 'Overlay', 'Brand', 'Interactive'],
      )}
    </ComponentSection>
  );
};

const TooltipsSection: React.FC = () => {
  const text = 'Tooltip text';
  const buttonWidth = 70;

  return (
    <ComponentSection id="Tooltips" title="Tooltips">
      <AntDCard>
        <p>
          A (<code>{'<Tooltip>'}</code>) is used to display a string value, and is triggered by
          interaction (either by click or hover) with a child element (usually a Button).
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Tooltip triggers</strong>
        <Space>
          <Tooltip content={text}>
            <Button>Trigger on hover</Button>
          </Tooltip>
          <Tooltip content={text} trigger="click">
            <Button>Trigger on click</Button>
          </Tooltip>
          <Tooltip content={text} trigger="contextMenu">
            <Button>Trigger on right click</Button>
          </Tooltip>
        </Space>
        <strong>Variations</strong>
        <p>Without arrow</p>
        <Space>
          <Tooltip content={text} placement="bottom" showArrow={false}>
            <Button>Tooltip without arrow</Button>
          </Tooltip>
        </Space>
        <p>Placement</p>
        <div>
          <div style={{ marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip content={text} placement="topLeft">
              <Button>TL</Button>
            </Tooltip>
            <Tooltip content={text} placement="top">
              <Button>Top</Button>
            </Tooltip>
            <Tooltip content={text} placement="topRight">
              <Button>TR</Button>
            </Tooltip>
          </div>
          <div style={{ float: 'left', width: buttonWidth }}>
            <Tooltip content={text} placement="leftTop">
              <Button>LT</Button>
            </Tooltip>
            <Tooltip content={text} placement="left">
              <Button>Left</Button>
            </Tooltip>
            <Tooltip content={text} placement="leftBottom">
              <Button>LB</Button>
            </Tooltip>
          </div>
          <div style={{ marginLeft: buttonWidth * 4 + 24, width: buttonWidth }}>
            <Tooltip content={text} placement="rightTop">
              <Button>RT</Button>
            </Tooltip>
            <Tooltip content={text} placement="right">
              <Button>Right</Button>
            </Tooltip>
            <Tooltip content={text} placement="rightBottom">
              <Button>RB</Button>
            </Tooltip>
          </div>
          <div style={{ clear: 'both', marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip content={text} placement="bottomLeft">
              <Button>BL</Button>
            </Tooltip>
            <Tooltip content={text} placement="bottom">
              <Button>Bottom</Button>
            </Tooltip>
            <Tooltip content={text} placement="bottomRight">
              <Button>BR</Button>
            </Tooltip>
          </div>
        </div>
      </AntDCard>
    </ComponentSection>
  );
};

const ColumnsSection: React.FC = () => {
  return (
    <ComponentSection id="Columns" title="Columns">
      <AntDCard>
        <p>
          The <code>{'<Columns>'}</code> component wraps child components to be displayed in
          multiple columns.
          <br />
          The <code>{'<Column>'}</code> component can optionally be used to wrap the content for
          each column and set its alignment.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <p>
          With <code>{'<Columns>'}</code> wrapper only, and <code>{'gap'}</code> set to 8 (default):
        </p>
        <Columns>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
        </Columns>
        <p>
          With <code>{'gap'}</code> set to 0:
        </p>
        <Columns gap={0}>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
        </Columns>
        <p>
          With <code>{'gap'}</code> set to 16:
        </p>
        <Columns gap={16}>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
          <Card>{loremIpsum}</Card>
        </Columns>
        <p>
          With left-aligned <code>{'<Column>'}</code>s (default):
        </p>
        <Columns>
          <Column>
            <Button>Content</Button>
          </Column>
          <Column>
            <Button>Content</Button>
          </Column>
          <Column>
            <Button>Content</Button>
          </Column>
        </Columns>
        <p>
          With center-aligned <code>{'<Column>'}</code>s:
        </p>
        <Columns>
          <Column align="center">
            <Button>Content</Button>
          </Column>
          <Column align="center">
            <Button>Content</Button>
          </Column>
          <Column align="center">
            <Button>Content</Button>
          </Column>
        </Columns>
        <p>
          With right-aligned <code>{'<Column>'}</code>s:
        </p>
        <Columns>
          <Column align="right">
            <Button>Content</Button>
          </Column>
          <Column align="right">
            <Button>Content</Button>
          </Column>
          <Column align="right">
            <Button>Content</Button>
          </Column>
        </Columns>
        <p>
          Variant with <code>{'page'}</code> prop, with margins and wrapping behavior, used for
          page-level layouts/headers:
        </p>
        <Columns page>
          <Column>
            <Button>Content 1</Button>
            <Button>Content 2</Button>
            <Button>Content 3</Button>
          </Column>
          <Column>
            <Button>Content 1</Button>
            <Button>Content 2</Button>
            <Button>Content 3</Button>
          </Column>
          <Column>
            <Button>Content 1</Button>
            <Button>Content 2</Button>
            <Button>Content 3</Button>
          </Column>
        </Columns>
      </AntDCard>
    </ComponentSection>
  );
};

const IconsSection: React.FC = () => {
  return (
    <ComponentSection id="Icons" title="Icons">
      <AntDCard>
        <p>
          An <code>{'<Icon>'}</code> component displays an icon from a custom font along with an
          optional tooltip.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Icon default</strong>
        <Icon name="star" title="star" />
        <strong>Icon variations</strong>
        <p>Icon with tooltip</p>
        <Icon name="star" title="Tooltip" />
        <p>Icon sizes</p>
        <Space wrap>
          {IconSizeArray.map((size) => (
            <Icon key={size} name="star" showTooltip size={size} title={size} />
          ))}
        </Space>
        <p>All icons</p>
        <Space wrap>
          {IconNameArray.map((name) => (
            <Icon key={name} name={name} showTooltip title={name} />
          ))}
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const ToastSection: React.FC = () => {
  return (
    <ComponentSection id="Toast" title="Toast">
      <AntDCard>
        <p>
          A <code>{'<Toast>'}</code> component is used to display a notification message at the
          viewport. Typically it&apos;s a notification providing a feedback based on the user
          interaction.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Default toast</strong>
        <Space>
          <Button
            onClick={() =>
              makeToast({
                description: 'Some informative content.',
                severity: 'Info',
                title: 'Default notification',
              })
            }>
            Open a default toast
          </Button>
        </Space>
        <strong>Variations</strong>
        <Space>
          <Button
            onClick={() =>
              makeToast({
                description: "You've triggered an error.",
                severity: 'Error',
                title: 'Error notification',
              })
            }>
            Open an error toast
          </Button>
          <Button
            onClick={() =>
              makeToast({
                description: "You've triggered an warning.",
                severity: 'Warning',
                title: 'Warning notification',
              })
            }>
            Open an warning toast
          </Button>
          <Button
            onClick={() =>
              makeToast({
                description: 'Action succed.',
                severity: 'Confirm',
                title: 'Success notification',
              })
            }>
            Open an success toast
          </Button>
        </Space>
        <Space>
          <Button
            onClick={() =>
              makeToast({
                closeable: false,
                description: "You've triggered an error.",
                severity: 'Error',
                title: 'Error notification',
              })
            }>
            Open a non-closable toast
          </Button>
          <Button
            onClick={() =>
              makeToast({
                description: 'Click below to design kit page.',
                link: <KitLink>View Design Kit</KitLink>,
                severity: 'Info',
                title: 'Welcome to design kit',
              })
            }>
            Open a toast with link
          </Button>
          <Button onClick={() => makeToast({ severity: 'Info', title: 'Compact notification' })}>
            Open a toast without description
          </Button>
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const ToggleSection: React.FC = () => {
  return (
    <ComponentSection id="Toggle" title="Toggle">
      <AntDCard>
        <p>
          A <code>{'<Toggle>'}</code> component represents switching between two states. This
          component is controlled by its parent and may optionally include a label.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Toggle default</strong>
        <Toggle />
        <strong>Toggle variations</strong>
        <Toggle checked={true} />
        <Toggle label="Label" />
      </AntDCard>
    </ComponentSection>
  );
};

/* modal section */

const handleSubmit = async (fail?: boolean) => {
  if (fail) throw new Error('Error message');
  await new Promise((r) => setTimeout(r, 1000));
  return;
};

const SmallModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal size="small" title={value}>
      <div>{value}</div>
    </Modal>
  );
};

const MediumModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal size="medium" title={value}>
      <div>{value}</div>
    </Modal>
  );
};

const LargeModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal size="large" title={value}>
      <div>{value}</div>
    </Modal>
  );
};

const IconModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal icon="experiment" title={value}>
      <div>{value}</div>
    </Modal>
  );
};

const LinksModalComponent: React.FC<{ value: string }> = ({ value }) => {
  return (
    <Modal cancel footerLink={<a>Footer Link</a>} headerLink={<a>Header Link</a>} title={value}>
      <div>{value}</div>
    </Modal>
  );
};

const FormModalComponent: React.FC<{ value: string; fail?: boolean }> = ({ value, fail }) => {
  return (
    <Modal
      cancel
      submit={{
        handleError,
        handler: () => handleSubmit(fail),
        text: 'Submit',
      }}
      title={value}>
      <Form>
        <Form.Item label="Workspace" name="workspaceId">
          <Select allowClear defaultValue={1} placeholder="Workspace (required)">
            <Option key="1" value="1">
              WS AS
            </Option>
            <Option key="2" value="2">
              Further
            </Option>
            <Option key="3" value="3">
              Whencelan
            </Option>
          </Select>
        </Form.Item>
        <Form.Item className={css.line} label="Template" name="template">
          <Select allowClear placeholder="No template (optional)">
            <Option key="1" value={1}>
              Default Template
            </Option>
          </Select>
        </Form.Item>
        <Form.Item className={css.line} label="Name" name="name">
          <Input defaultValue={value} placeholder="Name (optional)" />
        </Form.Item>
        <Form.Item className={css.line} label="Resource Pool" name="pool">
          <Select allowClear placeholder="Pick the best option">
            <Option key="1" value="1">
              GPU Pool
            </Option>
            <Option key="2" value="2">
              Aux Pool
            </Option>
          </Select>
        </Form.Item>
        <Form.Item className={css.line} label="Slots" name="slots">
          <InputNumber max={10} min={0} />
        </Form.Item>
      </Form>
    </Modal>
  );
};

const ValidationModalComponent: React.FC<{ value: string }> = ({ value }) => {
  const [form] = Form.useForm();
  const alias = Form.useWatch('alias', form);

  return (
    <Modal
      cancel
      submit={{
        disabled: !alias,
        handleError,
        handler: handleSubmit,
        text: 'Submit',
      }}
      title={value}>
      <Form form={form}>
        <Form.Item className={css.line} label="Name" name="name">
          <Input defaultValue={value} placeholder="Name (optional)" />
        </Form.Item>
        <Form.Item className={css.line} label="Alias" name="alias" required>
          <Input placeholder="Alias" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

const ModalSection: React.FC = () => {
  const [text, setText] = useState('State value that gets passed to modal via props');
  const SmallModal = useModal(SmallModalComponent);
  const MediumModal = useModal(MediumModalComponent);
  const LargeModal = useModal(LargeModalComponent);
  const FormModal = useModal(FormModalComponent);
  const FormFailModal = useModal(FormModalComponent);
  const LinksModal = useModal(LinksModalComponent);
  const IconModal = useModal(IconModalComponent);
  const ValidationModal = useModal(ValidationModalComponent);

  const confirm = useConfirm();
  const config = { content: text, title: text };
  const confirmDefault = () =>
    confirm({ ...config, onConfirm: voidPromiseFn, onError: handleError });
  const confirmDangerous = () =>
    confirm({
      ...config,
      danger: true,
      onConfirm: voidPromiseFn,
      onError: handleError,
    });

  return (
    <ComponentSection id="Modals" title="Modals">
      <AntDCard title="Usage">
        <Label>State value that gets passed to modal via props</Label>
        <Input value={text} onChange={(s) => setText(String(s.target.value))} />
        <hr />
        <strong>Sizes</strong>
        <Space>
          <Button onClick={SmallModal.open}>Open Small Modal</Button>
          <Button onClick={MediumModal.open}>Open Medium Modal</Button>
          <Button onClick={LargeModal.open}>Open Large Modal</Button>
        </Space>
        <hr />
        <strong>Links and Icons</strong>
        <Space>
          <Button onClick={LinksModal.open}>Open Modal with Header and Footer Links</Button>
          <Button onClick={IconModal.open}>Open Modal with Title Icon</Button>
        </Space>
        <hr />
        <strong>With form submission</strong>
        <Space>
          <Button onClick={FormModal.open}>Open Form Modal (Success)</Button>
          <Button onClick={FormFailModal.open}>Open Form Modal (Failure)</Button>
        </Space>
        <hr />
        <strong>With form validation</strong>
        <Space>
          <Button onClick={ValidationModal.open}>Open Modal with Form Validation</Button>
        </Space>
        <hr />
        <strong>Variations</strong>
        <Space>
          <Button onClick={confirmDefault}>Open Confirmation</Button>
          <Button onClick={confirmDangerous}>Open Dangerous Confirmation</Button>
        </Space>
      </AntDCard>
      <SmallModal.Component value={text} />
      <MediumModal.Component value={text} />
      <LargeModal.Component value={text} />
      <FormModal.Component value={text} />
      <FormFailModal.Component fail value={text} />
      <LinksModal.Component value={text} />
      <IconModal.Component value={text} />
      <ValidationModal.Component value={text} />
    </ComponentSection>
  );
};

const LongLoadingComponent = () => {
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    let active = true;
    setTimeout(() => {
      if (active) setLoaded(true);
    }, 5000);
    return () => {
      active = false;
    };
  }, []);

  return <div>This component is {loaded ? 'done loading!!!!!! wowza!!' : 'not loaded :('}</div>;
};

const AccordionSection: React.FC = () => {
  const [controlStateSingle, setControlStateSingle] = useState(false);
  const [controlStateGroup, setControlStateGroup] = useState(1);
  return (
    <ComponentSection id="Accordion" title="Accordion">
      <AntDCard>
        <p>
          An <code>{'<Accordion>'}</code> hides content behind a header. Typically found in forms,
          they hide complex content until the user interacts with the header.
        </p>
      </AntDCard>
      <AntDCard title="Singular usage">
        <p>
          An <code>{'<Accordion>'}</code> requires a title and content to show:
        </p>
        <Accordion title="Title">Children</Accordion>
        <p>
          By default, <code>{'<Accordion>'}</code> components control their open state themselves,
          but can be controlled externally:
        </p>
        <Checkbox
          checked={controlStateSingle}
          onChange={(e) => setControlStateSingle(e.target.checked)}>
          Check me to open the accordion below!
        </Checkbox>
        <Accordion open={controlStateSingle} title="Controlled by the above checkbox">
          Hello!
        </Accordion>
        <p>You can also render an uncontrolled accordion as open by default:</p>
        <Accordion defaultOpen title="Open by default">
          You should see me on page load.
        </Accordion>
        <p>
          By default, the content of an <code>{'<Accordion>'}</code> isn&apos;t mounted until
          opened, after which, the content stays mounted:
        </p>
        <Accordion title="Child will mount when opened and stay mounted after close">
          <LongLoadingComponent />
        </Accordion>
        <p>
          This can be changed to either mount the content along with the rest of the{' '}
          <code>{'<Accordion>'}</code> or to mount the content each time the component is opened:
        </p>
        <Accordion mountChildren="immediately" title="Child is already mounted">
          <LongLoadingComponent />
        </Accordion>
        <Accordion
          mountChildren="on-open"
          title="Child will mount when opened and unmount on close">
          <LongLoadingComponent />
        </Accordion>
      </AntDCard>
      <AntDCard title="Group usage">
        <p>
          <code>{'<Accordion>'}</code> components can be grouped together:
        </p>
        <Accordion.Group>
          <Accordion title="First child">One</Accordion>
          <Accordion title="Second child">Two</Accordion>
          <Accordion title="Third child">Three</Accordion>
        </Accordion.Group>
        <p>
          When grouped, the <code>{'<Accordion.Group>'}</code> component is responsible for keeping
          track of which component is open. As before, by default, the component keeps its own
          internal state, but can be controlled externally, as well as with a default initial state.
        </p>
        <Select value={controlStateGroup} onChange={(e) => setControlStateGroup(e as number)}>
          <Option key={1} value={1}>
            One
          </Option>
          <Option key={2} value={2}>
            Two
          </Option>
          <Option key={3} value={3}>
            Three
          </Option>
        </Select>
        <Accordion.Group openKey={controlStateGroup}>
          <Accordion key={1} title="First child">
            One
          </Accordion>
          <Accordion key={2} title="Second child">
            Two
          </Accordion>
          <Accordion key={3} title="Third child">
            Three
          </Accordion>
        </Accordion.Group>
        <Accordion.Group defaultOpenKey={3}>
          <Accordion key={1} title="First child">
            One
          </Accordion>
          <Accordion key={2} title="Second child">
            Two
          </Accordion>
          <Accordion key={3} title="Third child">
            Three! I&apos;m open by default!
          </Accordion>
        </Accordion.Group>
        <p>
          Controlled/uncontrolled <code>{'<Accordion.Group>'}</code> components can have multiple
          components open at the same time by default as well:
        </p>
        <Accordion.Group defaultOpenKey={[1, 3]}>
          <Accordion key={1} title="First child">
            One! I&apos;m open by default!
          </Accordion>
          <Accordion key={2} title="Second child">
            Two
          </Accordion>
          <Accordion key={3} title="Third child">
            Three! I&apos;m also open by default.
          </Accordion>
        </Accordion.Group>
        <p>
          You can configure an uncontrolled <code>{'<Accordion.Group>'}</code>
          component to only be able to have one child open at a time
        </p>
        <Accordion.Group exclusive>
          <Accordion key={1} title="First child">
            One! I&apos;m open by default!
          </Accordion>
          <Accordion key={2} title="Second child">
            Two
          </Accordion>
          <Accordion key={3} title="Third child">
            Three! I&apos;m also open by default.
          </Accordion>
        </Accordion.Group>
      </AntDCard>
    </ComponentSection>
  );
};

const DrawerSection: React.FC = () => {
  const [openLeft, setOpenLeft] = useState(false);
  const [openRight, setOpenRight] = useState(false);
  const scrollLines = [];
  for (let i = 0; i < 100; i++) {
    scrollLines.push(i);
  }

  return (
    <ComponentSection id="Drawer" title="Drawer">
      <AntDCard>
        <p>
          An <code>{'<Drawer>'}</code> is a full-height overlaid sidebar which moves into the
          viewport from the left or right side.
        </p>
      </AntDCard>
      <AntDCard title="Left side">
        <p>
          Drawer appears from the left side in an animation. Similar to a Modal, it can be closed
          only by clicking a Close button (at top right) or Escape key.
        </p>
        <p>If the drawer body has extra content, it is scrollable without hiding the header.</p>
        <Space>
          <Button onClick={() => setOpenLeft(true)}>Open Drawer</Button>
        </Space>
        <Drawer
          open={openLeft}
          placement="left"
          title="Left Drawer"
          onClose={() => setOpenLeft(!openLeft)}>
          {scrollLines.map((i) => (
            <p key={i}>Sample scrollable content</p>
          ))}
        </Drawer>
      </AntDCard>
      <AntDCard title="Right side">
        <p>Drawer appears from the right side.</p>
        <p>
          When a drawer has stateful content, that state is persisted when closed and re-opened.
        </p>
        <Space>
          <Button onClick={() => setOpenRight(true)}>Open Drawer</Button>
        </Space>
        <Drawer
          open={openRight}
          placement="right"
          title="Right Drawer"
          onClose={() => setOpenRight(!openRight)}>
          <p>Sample content</p>
          <Checkbox>A</Checkbox>
          <Checkbox>B</Checkbox>
          <Checkbox>C</Checkbox>
          <Checkbox>D</Checkbox>
          <Form.Item label="Sample Persistent Input" name="sample_drawer">
            <Input.TextArea />
          </Form.Item>
        </Drawer>
      </AntDCard>
    </ComponentSection>
  );
};

const SpinnerSection = () => {
  const [spinning, setSpinning] = useState(true);
  const [loadableData, setLoadableData] = useState<Loadable<string>>(NotLoaded);

  useEffect(() => {
    if (Loadable.isLoaded(loadableData)) return;
    let active = true;
    setTimeout(() => {
      if (active) setLoadableData(Loaded('This text has been loaded!'));
    }, 1000);
    return () => {
      active = false;
    };
  }, [loadableData]);

  return (
    <ComponentSection id="Spinner" title="Spinner">
      <AntDCard>
        <Paragraph>
          A <code>{'<Spinner>'}</code> indicates a loading state of a page or section.
        </Paragraph>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Spinner default</strong>
        <Spinner spinning />
        <strong>Spinner with children</strong>
        <div style={{ border: '1px solid var(--theme-surface-border)', padding: 8, width: '100%' }}>
          <Spinner spinning>
            <Card.Group size="medium">
              <Card size="medium" />
              <Card size="medium" />
            </Card.Group>
          </Spinner>
        </div>
        <strong>Spinner with conditional rendering</strong>
        <Toggle checked={spinning} label="Loading" onChange={setSpinning} />
        <div
          style={{
            border: '1px solid var(--theme-surface-border)',
            height: 300,
            padding: 8,
            width: '100%',
          }}>
          <Spinner conditionalRender spinning={spinning}>
            <Card size="medium" />
          </Spinner>
        </div>
        <strong>Loadable spinner</strong>
        <Button onClick={() => setLoadableData(NotLoaded)}>Unload</Button>
        <Spinner data={loadableData}>{(data) => <Paragraph>{data}</Paragraph>}</Spinner>
        <hr />
        <Header>Variations</Header>
        <strong>Centered Spinner</strong>
        <div
          style={{ border: '1px solid var(--theme-surface-border)', height: 200, width: '100%' }}>
          <Spinner center spinning />
        </div>
        <strong>Spinner with tip</strong>
        <Spinner spinning tip="Tip" />
        <strong>Spinner sizes</strong>
        <Space>
          {IconSizeArray.map((size) => (
            <Spinner key={size} size={size} spinning tip={size} />
          ))}
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const MessageSection: React.FC = () => {
  return (
    <ComponentSection id="Message" title="Message">
      <AntDCard>
        <Paragraph>
          A <code>{'<Message>'}</code> displays persistent information related to the application
          state. Requires at least one of description or title. Optionally displays an action button
          and/or an icon.
        </Paragraph>
      </AntDCard>
      <AntDCard title="Usage">
        <Message
          action={<Button>Optional action button</Button>}
          description={
            <>
              Message description, with a <Link to="">link to more info</Link>
            </>
          }
          icon="info"
          title="Message title"
        />
      </AntDCard>
    </ComponentSection>
  );
};

const Components = {
  Accordion: <AccordionSection />,
  Avatar: <AvatarSection />,
  Breadcrumbs: <BreadcrumbsSection />,
  Buttons: <ButtonsSection />,
  Cards: <CardsSection />,
  Charts: <ChartsSection />,
  Checkboxes: <CheckboxesSection />,
  ClipboardButton: <ClipboardButtonSection />,
  CodeEditor: <CodeEditorSection />,
  Color: <ColorSection />,
  Columns: <ColumnsSection />,
  DatePicker: <DatePickerSection />,
  Drawer: <DrawerSection />,
  Dropdown: <DropdownSection />,
  Form: <FormSection />,
  Icons: <IconsSection />,
  InlineForm: <InlineFormSection />,
  Input: <InputSection />,
  InputNumber: <InputNumberSection />,
  InputSearch: <InputSearchSection />,
  InputShortcut: <InputShortcutSection />,
  Lists: <ListsSection />,
  LogViewer: <LogViewerSection />,
  Message: <MessageSection />,
  Modals: <ModalSection />,
  Nameplate: <NameplateSection />,
  Notes: <NotesSection />,
  Pagination: <PaginationSection />,
  Pivot: <PivotSection />,
  Select: <SelectSection />,
  Spinner: <SpinnerSection />,
  Tags: <TagsSection />,
  Theme: <ThemeSection />,
  Toast: <ToastSection />,
  Toggle: <ToggleSection />,
  Tooltips: <TooltipsSection />,
  Typography: <TypographySection />,
};

const DesignKit: React.FC = () => {
  const { actions } = useUI();
  const location = useLocation();
  const searchParams = new URLSearchParams(location.search);
  const isExclusiveMode = searchParams.get('exclusive') === 'true';
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);

  const closeDrawer = useCallback(() => {
    setIsDrawerOpen(false);
  }, []);

  useEffect(() => {
    actions.hideChrome();
  }, [actions]);

  useEffect(() => {
    // move to the specified anchor tag in the url after refreshing page
    if (window.location.hash) {
      const hashSave = window.location.hash;
      window.location.hash = ''; // clear hash first
      window.location.hash = hashSave; // set hash again
    }
  }, []);

  return (
    <Page bodyNoPadding breadcrumb={[]} docTitle="Design Kit" stickyHeader>
      <div className={css.base}>
        <nav className={css.default}>
          <Link reloadDocument to={'/'}>
            <Logo branding={BrandingType.Determined} orientation="horizontal" />
          </Link>
          <ThemeToggle />
          <ul className={css.sections}>
            {componentOrder.map((componentId) => (
              <li key={componentId}>
                <a href={`#${componentId}`}>{ComponentTitles[componentId]}</a>
              </li>
            ))}
          </ul>
        </nav>
        <nav className={css.mobile}>
          <Link reloadDocument to={'/'}>
            <Logo branding={BrandingType.Determined} orientation="horizontal" />
          </Link>
          <div className={css.controls}>
            <ThemeToggle iconOnly />
            <Button onClick={() => setIsDrawerOpen(true)}>Sections</Button>
          </div>
        </nav>
        <article>
          {componentOrder
            .filter((id) => !isExclusiveMode || !location.hash || id === location.hash.substring(1))
            .map((componentId) => (
              <React.Fragment key={componentId}>{Components[componentId]}</React.Fragment>
            ))}
        </article>
        <Drawer open={isDrawerOpen} placement="right" title="Sections" onClose={closeDrawer}>
          <ul className={css.sections}>
            {componentOrder.map((componentId) => (
              <li key={componentId} onClick={closeDrawer}>
                <a href={`#${componentId}`}>{ComponentTitles[componentId]}</a>
              </li>
            ))}
          </ul>
        </Drawer>
      </div>
    </Page>
  );
};

export default DesignKit;
