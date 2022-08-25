import { Tabs, TabsProps } from 'antd';
import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react';
import { useHistory, useParams } from 'react-router-dom';

import { isEqual } from 'shared/utils/data';

interface DynamicTabBarProps extends Omit<TabsProps, 'activeKey'> {
  basePath: string;
}

type TabBarUpdater = (node?: JSX.Element) => void;

const TabBarContext = createContext<TabBarUpdater | undefined>(undefined);

const DynamicTabs: React.FC<DynamicTabBarProps> = ({
  basePath,
  children,
  ...props
}): JSX.Element => {
  const [ tabBarExtraContent, setTabBarExtraContent ] = useState<JSX.Element | undefined>();

  const history = useHistory();

  const [ tabKeys, setTabKeys ] = useState<string[]>([]);

  useEffect(() => {
    const newTabKeys = React.Children.map(children, (c) => (c as {key: string})?.key ?? '');
    if (Array.isArray(newTabKeys) && !isEqual(newTabKeys, tabKeys)) setTabKeys(newTabKeys);
  }, [ children, tabKeys ]);

  const { tab } = useParams<{tab: string}>();

  const [ activeKey, setActiveKey ] = useState(tab);

  const handleTabSwitch = useCallback((key: string) => {

    history.push(`${basePath}/${key}`);
    setActiveKey(key);
  }, [ history, basePath ]);

  useEffect(() => { setActiveKey(tab); }, [ tab ]);

  useEffect(() => {

    if (!activeKey && tabKeys.length) {
      history.replace(`${basePath}/${tabKeys[0]}`);

    }
  }, [ activeKey, tabKeys, handleTabSwitch, basePath, history ]);

  const updateTabBarContent: TabBarUpdater = useCallback((content?: JSX.Element) => {
    // console.log(content);
    setTabBarExtraContent(content);
  }, []);

  return (
    <TabBarContext.Provider value={updateTabBarContent}>
      <Tabs
        {...props}
        activeKey={activeKey}
        tabBarExtraContent={tabBarExtraContent}
        onTabClick={handleTabSwitch}>
        {children}
      </Tabs>
    </TabBarContext.Provider>
  );
};

export default DynamicTabs;

export const useSetDynamicTabBar = (content: JSX.Element): void => {
  const updateTabBarContent = useContext(TabBarContext);
  if (!updateTabBarContent) console.error('must useSetDynamicTabBar within TabBarContext');
  useEffect(() => {
    updateTabBarContent?.(content);
    // return () => updateTabBarContent(undefined);
  }, [ updateTabBarContent, content ]);
};
