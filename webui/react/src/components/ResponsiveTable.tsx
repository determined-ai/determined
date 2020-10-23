import { Table } from 'antd';
import { TableProps } from 'antd/es/table';
import React, { useEffect, useRef, useState } from 'react';

import useResize from 'hooks/useResize';

/* eslint-disable-next-line @typescript-eslint/ban-types */
type ResponsiveTable = <T extends object>(props: TableProps<T>) => JSX.Element;

const ResponsiveTable: ResponsiveTable = ({ scroll, ...props }) => {
  const tableRef = useRef<HTMLDivElement>(null);
  const [ tableScroll, setTableScroll ] = useState(scroll);
  const resize = useResize(tableRef);

  useEffect(() => {
    if (!tableRef.current) return;

    const tables = tableRef.current.getElementsByTagName('table');
    if (tables.length === 0) return;

    const rect = tables[0].getBoundingClientRect();
    setTableScroll({
      x: rect.width > resize.width ? rect.width : scroll?.x,
      y: scroll?.y,
    });
  }, [ resize, scroll ]);

  return <div ref={tableRef}>
    <Table scroll={tableScroll} {...props} />
  </div>;
};

export default ResponsiveTable;
