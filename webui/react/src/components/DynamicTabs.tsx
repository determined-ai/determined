import Pivot, { PivotProps, PivotTabType } from 'hew/Pivot';
import _ from 'lodash';
import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

interface DynamicTabBarProps extends Omit<PivotProps, 'activeKey' | 'type'> {
  basePath: string;
  type?: PivotTabType;
}

type TabBarUpdater = (node?: JSX.Element) => void;

const TabBarContext = createContext<TabBarUpdater | undefined>(undefined);

const DynamicTabs: React.FC<DynamicTabBarProps> = ({ basePath, items, ...props }): JSX.Element => {
  const [tabBarExtraContent, setTabBarExtraContent] = useState<JSX.Element | undefined>();

  const navigate = useNavigate();

  const [tabKeys, setTabKeys] = useState<string[]>([]);

  useEffect(() => {
    const newTabKeys = items?.map((c) => c.key ?? '');
    if (Array.isArray(newTabKeys) && !_.isEqual(newTabKeys, tabKeys)) setTabKeys(newTabKeys);
  }, [items, tabKeys]);

  const { tab } = useParams<{ tab: string }>();

  const [activeKey, setActiveKey] = useState(tab);

  const handleTabSwitch = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`);
      setActiveKey(key);
    },
    [navigate, basePath],
  );

  useEffect(() => {
    setActiveKey(tab);
  }, [tab]);

  useEffect(() => {
    if ((!activeKey || !tabKeys.includes(activeKey)) && tabKeys.length) {
      navigate(`${basePath}/${tabKeys[0]}`, { replace: true });
    }
  }, [activeKey, tabKeys, handleTabSwitch, basePath, navigate]);

  const updateTabBarContent: TabBarUpdater = useCallback((content?: JSX.Element) => {
    setTabBarExtraContent(content);
  }, []);

  return (
    <TabBarContext.Provider value={updateTabBarContent}>
      <Pivot
        {...props}
        activeKey={activeKey}
        items={items}
        tabBarExtraContent={tabBarExtraContent}
        onTabClick={handleTabSwitch}
      />
    </TabBarContext.Provider>
  );
};

export default DynamicTabs;

export const useSetDynamicTabBar = (content: JSX.Element | undefined): void => {
  const updateTabBarContent = useContext(TabBarContext);
  useEffect(() => {
    if (content !== undefined) updateTabBarContent?.(content);

    return () => {
      updateTabBarContent?.(undefined);
    };
  }, [updateTabBarContent, content]);
};
