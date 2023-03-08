import { PoweroffOutlined } from '@ant-design/icons';
import { Card as AntDCard, Space } from 'antd';
import { LabeledValue, SelectValue } from 'antd/es/select';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';

import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import Card from 'components/kit/Card';
import Checkbox from 'components/kit/Checkbox';
import Empty from 'components/kit/Empty';
import Facepile from 'components/kit/Facepile';
import Form from 'components/kit/Form';
import IconicButton from 'components/kit/IconicButton';
import Input from 'components/kit/Input';
import InputNumber from 'components/kit/InputNumber';
import InputSearch from 'components/kit/InputSearch';
import { LineChart, Serie } from 'components/kit/LineChart';
import { useChartGrid } from 'components/kit/LineChart/useChartGrid';
import { XAxisDomain } from 'components/kit/LineChart/XAxisFilter';
import LogViewer from 'components/kit/LogViewer/LogViewer';
import Pagination from 'components/kit/Pagination';
import Pivot from 'components/kit/Pivot';
import Select from 'components/kit/Select';
import Toggle from 'components/kit/Toggle';
import Tooltip from 'components/kit/Tooltip';
import UserAvatar from 'components/kit/UserAvatar';
import UserBadge from 'components/kit/UserBadge';
import { useTags } from 'components/kit/useTags';
import Logo from 'components/Logo';
import OverviewStats from 'components/OverviewStats';
import Page from 'components/Page';
import ProjectCard from 'components/ProjectCard';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResponsiveTable from 'components/Table/ResponsiveTable';
import ThemeToggle from 'components/ThemeToggle';
import { drawPointsPlugin } from 'components/UPlot/UPlotChart/drawPointsPlugin';
import { tooltipsPlugin } from 'components/UPlot/UPlotChart/tooltipsPlugin2';
import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { V1LogLevel } from 'services/api-ts-sdk';
import { mapV1LogsResponse } from 'services/decoder';
import useUI from 'shared/contexts/stores/UI';
import { ValueOf } from 'shared/types';
import { noOp } from 'shared/utils/service';
import {
  generateTestProjectData,
  generateTestWorkspaceData,
} from 'storybook/shared/generateTestData';
import { BrandingType, MetricType, Project, ResourcePool, User } from 'types';

import css from './DesignKit.module.scss';
import { CheckpointsDict } from './TrialDetails/F_TrialDetailsOverview';
import WorkspaceCard from './WorkspaceList/WorkspaceCard';

const ComponentTitles = {
  Breadcrumbs: 'Breadcrumbs',
  Buttons: 'Buttons',
  Cards: 'Cards',
  Charts: 'Charts',
  Checkboxes: 'Checkboxes',
  Empty: 'Empty',
  Facepile: 'Facepile',
  Form: 'Form',
  Input: 'Input',
  InputNumber: 'InputNumber',
  InputSearch: 'InputSearch',
  Lists: 'Lists (tables)',
  LogViewer: 'LogViewer',
  Pagination: 'Pagination',
  Pivot: 'Pivot',
  Select: 'Select',
  Tags: 'Tags',
  Toggle: 'Toggle',
  Tooltips: 'Tooltips',
  UserAvatar: 'UserAvatar',
  UserBadge: 'UserBadge',
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
    <section>
      <h3 id={id}>{title}</h3>
      {children}
    </section>
  );
};

const ButtonsSection: React.FC = () => {
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
        <strong>Default Button</strong>
        <Space>
          <Button type="primary">Primary</Button>
          <Button>Secondary</Button>
          <Button loading>Loading</Button>
          <Button disabled>Disabled</Button>
        </Space>
        <hr />
        <strong>Default Button with icon</strong>
        <Space>
          <Button icon={<PoweroffOutlined />} type="primary">
            ButtonWithIcon
          </Button>
          <Button icon={<PoweroffOutlined />}>ButtonWithIcon</Button>
          <Button disabled icon={<PoweroffOutlined />}>
            ButtonWithIcon
          </Button>
        </Space>
        <hr />
        <strong>Large iconic buttons</strong>
        <Space>
          <IconicButton iconName="searcher-grid" text="Iconic button" type="primary" />
          <IconicButton iconName="searcher-grid" text="Iconic button" />
          <IconicButton disabled iconName="searcher-grid" text="Iconic button" />
        </Space>
      </AntDCard>
    </ComponentSection>
  );
};

const SelectSection: React.FC = () => {
  const handleFilter = (input: string, option: LabeledValue | undefined) =>
    !!(option?.label && option.label.toString().includes(input) === true);
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
          filterOption={handleFilter}
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
          filterSort={(a: LabeledValue, b: LabeledValue) =>
            (a?.label ? a.label : 0) > (b?.label ? b?.label : 0) ? 1 : -1
          }
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
          Also see{' '}
          <Link reloadDocument to={`#${ComponentTitles.Form}`}>
            Form
          </Link>{' '}
          for form-specific variations
        </span>
      </AntDCard>
    </ComponentSection>
  );
};

const line1BatchesDataRaw: [number, number][] = [
  [0, -2],
  [2, Math.random() * 12],
  [4, 15],
  [6, Math.random() * 60],
  [9, Math.random() * 40],
  [10, Math.random() * 76],
  [18, Math.random() * 80],
  [19, 89],
];
const line2BatchesDataRaw: [number, number][] = [
  [1, 15],
  [2, 10.123456789],
  [2.5, Math.random() * 22],
  [3, 10.3909],
  [3.25, 19],
  [3.75, 4],
  [4, 12],
];

const ChartsSection: React.FC = () => {
  const timerRef = useRef<NodeJS.Timer | null>(null);
  const [timer, setTimer] = useState(1);
  useEffect(() => {
    timerRef.current = setInterval(() => setTimer((t) => t + 1), 2000);

    return () => {
      if (timerRef.current !== null) clearInterval(timerRef.current);
    };
  }, []);

  const line1BatchesDataStreamed = useMemo(() => line1BatchesDataRaw.slice(0, timer), [timer]);
  const line2BatchesDataStreamed = useMemo(() => line2BatchesDataRaw.slice(0, timer), [timer]);

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
        <LineChart height={250} series={[line1, line2]} showLegend={true} title="Sample" />
      </AntDCard>
      <AntDCard title="Focus series">
        <p>Highlight a specific metric in the chart.</p>
        <LineChart focusedSeries={1} height={250} series={[line1, line2]} title="Sample" />
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
          Also see{' '}
          <Link reloadDocument to={`#${ComponentTitles.Form}`}>
            Form
          </Link>{' '}
          for form-specific variations
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
          Also see{' '}
          <Link reloadDocument to={`#${ComponentTitles.Form}`}>
            Form
          </Link>{' '}
          for form-specific variations
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

const BreadcrumbsSection: React.FC = () => {
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
            of a page.{' '}
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
      </AntDCard>
    </ComponentSection>
  );
};

const FacepileSection: React.FC = () => {
  const users = [
    {
      id: 123,
      isActive: true,
      isAdmin: true,
      username: 'Fake Admin',
    },
    {
      id: 3,
      isActive: true,
      isAdmin: true,
      username: 'Admin',
    },
    {
      id: 13,
      isActive: true,
      isAdmin: true,
      username: 'Fake',
    },
    {
      id: 23,
      isActive: true,
      isAdmin: true,
      username: 'User',
    },
    {
      id: 12,
      isActive: true,
      isAdmin: true,
      username: 'Foo',
    },
    {
      id: 2,
      isActive: true,
      isAdmin: true,
      username: 'Baar',
    },
    {
      id: 12,
      isActive: true,
      isAdmin: true,
      username: 'Gandalf',
    },
    {
      id: 1,
      isActive: true,
      isAdmin: true,
      username: 'Leroy Jenkins',
    },
  ];
  return (
    <ComponentSection id="Facepile" title="Facepile">
      <AntDCard>
        <p>
          A face pile (<code>{'<Facepile>'}</code>) displays a list of personas. Each circle
          represents a person and contains their image or initials. Often this control is used when
          sharing who has access to a specific view or file, or when assigning someone a task within
          a workflow.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content considerations</strong>
        <ul>
          <li>
            The face pile empty state should only include an &quot;Add&quot; button. Another variant
            is to use an input field with placeholder text instructing people to add a person. See
            the people picker component for the menu used to add people to the face pile list.
          </li>
          <li>
            When there is only one person in the face pile, consider using their name next to the
            face or initials.
          </li>
          <li>
            When there is a need to show the face pile expanded into a vertical list, include a
            downward chevron button. Selecting the chevron opens a standard list view of people.
          </li>
          <li>
            When the face pile exceeds a max number of 5 people, show a button at the end of the
            list indicating how many are not being shown. Clicking or tapping on the overflow would
            open a standard list view of people.
          </li>
          <li>
            The component can include an &quot;Add&quot; button which can be used for quickly adding
            a person to the list.
          </li>
          <li>
            When hovering over a person in the face pile, include a tooltip or people card that
            offers more information about that person.
          </li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Facepile with initial state</strong>
        <Facepile editable selectableUsers={users} />
        <strong>Variations</strong>
        <ul>
          <li>
            Facepile with 8 people
            <Facepile users={users.slice(0, 8)} />
          </li>
          <li>Facepile with both name initials</li>
          <p>Check the Facepile above and select a user that would fit that case</p>
        </ul>
      </AntDCard>
    </ComponentSection>
  );
};

const UserAvatarSection: React.FC = () => {
  return (
    <ComponentSection id="UserAvatar" title="UserAvatar">
      <AntDCard>
        <p>
          A (<code>{'<UserAvatar>'}</code>) represents a user. It consists of a circle containing
          the first letter of the user&apos;s display name or username. On hover, it displays a
          tooltip with the full display name or username.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <UserAvatar />
      </AntDCard>
    </ComponentSection>
  );
};

const UserBadgeSection: React.FC = () => {
  const testUser = { displayName: 'Abc', id: 1, username: 'alpha123' };

  return (
    <ComponentSection id="UserBadge" title="UserBadge">
      <AntDCard>
        <p>
          A (<code>{'<UserBadge>'}</code>) fully represents a user with a UserAvatar circle icon,
          and the user&apos;s display name and username. If there is a display name, it appears
          first, otherwise only the username is visible. A &apos;compact&apos; option reduces the
          size of the name for use in a smaller form or modal.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <li>User with Display Name</li>
        <UserBadge user={testUser as User} />
        <li>Compact format</li>
        <UserBadge compact user={testUser as User} />
        <li>User without Display Name</li>
        <UserBadge user={{ ...testUser, displayName: undefined } as User} />
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
        <Pagination total={500} />
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
  const rps = resourcePools as unknown as ResourcePool[];
  const project: Project = { ...generateTestProjectData(), lastExperimentStartedAt: new Date() };
  const workspace = generateTestWorkspaceData();

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
          <Card actionMenu={{ items: [{ key: 'test', label: 'Test' }] }}>Card with actions</Card>
          <Card actionMenu={{ items: [{ key: 'test', label: 'Test' }] }} disabled>
            Disabled card
          </Card>
          <Card onClick={noOp}>Clickable card</Card>
        </Card.Group>
        <p>Medium cards</p>
        <Card.Group size="medium">
          <Card actionMenu={{ items: [{ key: 'test', label: 'Test' }] }} size="medium">
            Card with actions
          </Card>
          <Card actionMenu={{ items: [{ key: 'test', label: 'Test' }] }} disabled size="medium">
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
        <strong>Card examples</strong>
        <ul>
          <li>
            Project card (<code>{'<ProjectCard>'}</code>)
          </li>
          <Card.Group>
            <ProjectCard project={project} />
            <ProjectCard project={{ ...project, archived: true }} />
            <ProjectCard
              project={{
                ...project,
                name: 'Project with a very long name that spans many lines and eventually gets cut off',
              }}
            />
          </Card.Group>
          <li>
            Workspace card (<code>{'<WorkspaceCard>'}</code>)
          </li>
          <Card.Group size="medium">
            <WorkspaceCard workspace={workspace} />
            <WorkspaceCard workspace={{ ...workspace, archived: true }} />
          </Card.Group>
          <li>
            Stats overview (<code>{'<OverviewStats>'}</code>)
          </li>
          <Card.Group>
            <OverviewStats title="Active Experiments">0</OverviewStats>
            <OverviewStats title="Clickable card" onClick={noOp}>
              Example
            </OverviewStats>
          </Card.Group>
          <li>
            Resource pool card (<code>{'<ResourcePoolCard>'}</code>)
          </li>
          <Card.Group size="medium">
            <ResourcePoolCard resourcePool={rps[0]} />
          </Card.Group>
        </ul>
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
          <LogViewer decoder={mapV1LogsResponse} initialLogs={sampleLogs} sortKey="id" />
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
            Form-specific{' '}
            <Link reloadDocument to={`#${ComponentTitles.Input}`}>
              Input
            </Link>{' '}
            variations
          </strong>
          <br />
          <Form.Item label="Required input" name="required" required>
            <Input />
          </Form.Item>
          <Form.Item
            label="Invalid input"
            name="invalid"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific{' '}
            <Link reloadDocument to={`#${ComponentTitles.Input}`}>
              TextArea
            </Link>{' '}
            variations
          </strong>
          <br />
          <Form.Item label="Required TextArea" name="required" required>
            <Input.TextArea />
          </Form.Item>
          <Form.Item
            label="Invalid TextArea"
            name="invalid"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input.TextArea />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific{' '}
            <Link reloadDocument to={`#${ComponentTitles.Input}`}>
              Password
            </Link>{' '}
            variations
          </strong>
          <br />
          <Form.Item label="Required Password" name="required" required>
            <Input.Password />
          </Form.Item>
          <Form.Item
            label="Invalid Password"
            name="invalid"
            validateMessage="Input validation error"
            validateStatus="error">
            <Input.Password />
          </Form.Item>
          <br />
          <hr />
          <br />
          <strong>
            Form-specific{' '}
            <Link reloadDocument to={`#${ComponentTitles.InputNumber}`}>
              InputNumber
            </Link>{' '}
            variations
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
            Form-specific{' '}
            <Link reloadDocument to={`#${ComponentTitles.Select}`}>
              Select
            </Link>{' '}
            variations
          </strong>
          <Form.Item label="Required dropdown" name="required" required>
            <Select
              defaultValue={1}
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
      <Card>
        <p>
          The editable tags list (<code>{'<Tags>'}</code>) supports &quot;add&quot;,
          &quot;edit&quot; and &quot;remove&quot; actions on individual tags.
        </p>
      </Card>
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

const TooltipsSection: React.FC = () => {
  const text = 'Tooltip text';
  const buttonWidth = 70;

  return (
    <ComponentSection id="Tooltips" title="Tooltips">
      <AntDCard>
        <p>
          A good tooltip (<code>{'<Tooltip>'}</code>) briefly describes unlabeled controls or
          provides a bit of additional information about labeled controls, when this is useful. It
          can also help customers navigate the UI by offering additional—not redundant—information
          about control labels, icons, and links. A tooltip should always add valuable information;
          use sparingly.
        </p>
      </AntDCard>
      <AntDCard title="Best practices">
        <strong>Content</strong>
        <ul>
          <li>
            Don&apos;t use a tooltip to restate a button name that&apos;s already shown in the UI.
          </li>
          <li>
            When a control or UI element is unlabeled, use a simple, descriptive noun phrase. For
            Only use periods for complete sentences.italize the first word (unless a subsequent word
            is a proper noun), and don&apos;t use a period.
          </li>
          <li>
            For a disabled control that could use an explanation, provide a brief description of the
            state in which the control will be enabled. For example: “This feature is available for
            line charts.”
          </li>
          <li>Only use periods for complete sentences.</li>
        </ul>
      </AntDCard>
      <AntDCard title="Usage">
        <strong>Tooltips default</strong>
        <Space>
          <Tooltip title={text}>
            <span>Trigger on hover</span>
          </Tooltip>
          <Tooltip title={text} trigger="click">
            <span>Trigger on click</span>
          </Tooltip>
          <Tooltip title={text} trigger="contextMenu">
            <span>Trigger on right click</span>
          </Tooltip>
        </Space>
        <strong>Considerations</strong>
        <ul>
          <li>
            Nest the tooltip where the content in a cell/text is. Don’t let it levitate in the
            nothingness.
          </li>
        </ul>
        <strong>Variations</strong>
        <div>
          <div style={{ marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip placement="topLeft" title={text}>
              <Button>TL</Button>
            </Tooltip>
            <Tooltip placement="top" title={text}>
              <Button>Top</Button>
            </Tooltip>
            <Tooltip placement="topRight" title={text}>
              <Button>TR</Button>
            </Tooltip>
          </div>
          <div style={{ float: 'left', width: buttonWidth }}>
            <Tooltip placement="leftTop" title={text}>
              <Button>LT</Button>
            </Tooltip>
            <Tooltip placement="left" title={text}>
              <Button>Left</Button>
            </Tooltip>
            <Tooltip placement="leftBottom" title={text}>
              <Button>LB</Button>
            </Tooltip>
          </div>
          <div style={{ marginLeft: buttonWidth * 4 + 24, width: buttonWidth }}>
            <Tooltip placement="rightTop" title={text}>
              <Button>RT</Button>
            </Tooltip>
            <Tooltip placement="right" title={text}>
              <Button>Right</Button>
            </Tooltip>
            <Tooltip placement="rightBottom" title={text}>
              <Button>RB</Button>
            </Tooltip>
          </div>
          <div style={{ clear: 'both', marginLeft: buttonWidth, whiteSpace: 'nowrap' }}>
            <Tooltip placement="bottomLeft" title={text}>
              <Button>BL</Button>
            </Tooltip>
            <Tooltip placement="bottom" title={text}>
              <Button>Bottom</Button>
            </Tooltip>
            <Tooltip placement="bottomRight" title={text}>
              <Button>BR</Button>
            </Tooltip>
          </div>
        </div>
      </AntDCard>
    </ComponentSection>
  );
};

const EmptySection: React.FC = () => {
  return (
    <ComponentSection id="Empty" title="Empty">
      <AntDCard>
        <p>
          An <code>{'<Empty>'}</code> component indicates that no content is available for a page.
          It may display an icon and a description explaining why this state is displayed.
        </p>
      </AntDCard>
      <AntDCard title="Usage">
        <Empty
          description={
            <>
              Empty component description, with a <Link to="">link to more info</Link>
            </>
          }
          icon="warning-large"
          title="Empty title"
        />
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

const Components = {
  Breadcrumbs: <BreadcrumbsSection />,
  Buttons: <ButtonsSection />,
  Cards: <CardsSection />,
  Charts: <ChartsSection />,
  Checkboxes: <CheckboxesSection />,
  Empty: <EmptySection />,
  Facepile: <FacepileSection />,
  Form: <FormSection />,
  Input: <InputSection />,
  InputNumber: <InputNumberSection />,
  InputSearch: <InputSearchSection />,
  Lists: <ListsSection />,
  LogViewer: <LogViewerSection />,
  Pagination: <PaginationSection />,
  Pivot: <PivotSection />,
  Select: <SelectSection />,
  Tags: <TagsSection />,
  Toggle: <ToggleSection />,
  Tooltips: <TooltipsSection />,
  UserAvatar: <UserAvatarSection />,
  UserBadge: <UserBadgeSection />,
};

const DesignKit: React.FC = () => {
  const { actions } = useUI();

  useEffect(() => {
    actions.hideChrome();
  }, [actions]);

  return (
    <Page bodyNoPadding docTitle="Design Kit">
      <div className={css.base}>
        <nav>
          <Link reloadDocument to={{}}>
            <Logo branding={BrandingType.Determined} orientation="horizontal" />
          </Link>
          <ThemeToggle />
          <ul>
            {componentOrder.map((componentId) => (
              <li key={componentId}>
                <Link reloadDocument to={`#${componentId}`}>
                  {ComponentTitles[componentId]}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <main>{componentOrder.map((componentId) => Components[componentId])}</main>
      </div>
    </Page>
  );
};

export default DesignKit;
