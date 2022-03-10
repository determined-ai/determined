// @ts-nocheck
import React, { useState, useEffect } from 'react';
import ReactDOM from "react-dom";
import "antd/dist/antd.min.css";
import "./DragSortingTable.css";
import { Table } from "antd";
import { Resizable } from "react-resizable";
import ReactDragListView from "react-drag-listview";
import { processApiError } from 'services/utils';
import { DragOutlined  } from '@ant-design/icons';

const columns =  [
  {
    title: <span className="dragHandler">Key</span>,
    dataIndex: "id",
    render: (text) => <span>{text}</span>,
    width: 50
  },
  {
    title: <span className="dragHandler">Name</span>,
    dataIndex: "name",
    width: 200
  },
  {
    title: <span className="dragHandler">Forked From</span>,
    dataIndex: "forkedFrom",
    width: 100
  },
  {
    title: <span className="dragHandler">Num Trials</span>,
    dataIndex: "numTrials",
    width: 100
  },
  {
    title: <span className="dragHandler">Searcher Type</span>,
    dataIndex: "searcherType"
  }
]

const dataSource = [
  {
    key: "1",
    name: "Boran",
    gender: "male",
    age: "12",
    address: "New York"
  },
  {
    key: "2",
    name: "JayChou",
    gender: "male",
    age: "38",
    address: "TaiWan"
  },
  {
    key: "3",
    name: "Lee",
    gender: "female",
    age: "22",
    address: "BeiJing"
  },
  {
    key: "4",
    name: "ChouTan",
    gender: "male",
    age: "31",
    address: "HangZhou"
  },
  {
    key: "5",
    name: "AiTing",
    gender: "female",
    age: "22",
    address: "Xi’An"
  }
];

const ResizableTitle = ({ onResize, width, ...restProps }) => {
  if (!width) {
    return <th {...restProps} />;
  }

  return (
    <Resizable
      width={width}
      height={0}
      handle={
        <span
          className="react-resizable-handle"
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
      }
      onResize={onResize}
      draggableOpts={{ enableUserSelectHack: false }}
    >
      <th style={{
        // padding: 0
      }}>
      <DragOutlined />  <div {...restProps}/>
      </th>
    </Resizable>
  );
};

const Demo = ({
  dataSource,
  columns,
  ...props
}) => {
  // console.log({columns})
  const [data, setData] = useState(dataSource);
  const [columnOrder, setColumnOrder] = useState(columns);

  useEffect(() => {
    setData(dataSource);
  }, [dataSource]);

  // useEffect(() => {
  //   setColumnOrder(columns);
  // }, [columns]);

  const dragProps = {
    onDragEnd: (fromIndex, toIndex) => {
    
      const reorderedColumns = [...columnOrder];
      const item = reorderedColumns.splice(fromIndex- 1, 1)[0];
      console.log({  fromIndex, toIndex })
      reorderedColumns.splice(toIndex-1, 0, item);
      setColumnOrder(reorderedColumns);
    },
    //nodeSelector: 'th',
    //handleSelector: '.anticon-drag',
    // handleSelector: 'th',
    ignoreSelector: 'react-resizable-handle',
  };

  const components = {
    header: {
      cell: ResizableTitle,
    },
  };

  const handleResize = (index) => (e, { size }) => {
    setColumnOrder(
      columnOrder.map((col, i) => (index === i ? { ...col, width: size.width } : col))
    );
  };

  const renderColumns = columnOrder.map((col, index) => ({
    ...col,
    // onHeaderCell: (column) => ({
    //   width: column.width ?? 1,
    //   onResize: handleResize(index),
    // }),
  }));

  return (
    // <ReactDragListView.DragColumn {...dragProps}>
      <Table
        bordered
        // components={components}
        // columns={renderColumns}
        columns={columns}
        dataSource={data}
        // scroll={tableScroll}
        tableLayout="auto"
        {...props}
      />
    // </ReactDragListView.DragColumn>
  );
};


export default Demo