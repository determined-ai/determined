import useModal from "./useModal";
import UserSettings from "components/UserSettings";
import css from "./useModalUserSettings.module.scss";
import { useStore } from "contexts/Store";

const useModalUserSettings = () => {
  const { modalOpen } = useModal();
  const { auth } = useStore();
  const username = auth.user?.username || "Anonymous";

  const getModalContent = () => {
    return <UserSettings username={username} />;
  };

  const openUserSettingsModal = () => {
    modalOpen({
      content: getModalContent(),
      icon: null,
      className: css.noFooter,
      closable: true,
      title: "Account",
    });
  };

  return { openUserSettingsModal };
};

export default useModalUserSettings;
