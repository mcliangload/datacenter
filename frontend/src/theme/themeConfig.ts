import type { ThemeConfig } from 'antd'

const themeConfig: ThemeConfig = {
  token: {
    colorPrimary: '#1890ff',
    colorLink: '#1890ff',
    colorLinkHover: '#40a9ff',
    colorLinkActive: '#096dd9',
    colorSuccess: '#52c41a',
    colorWarning: '#faad14',
    colorError: '#f5222d',
    colorInfo: '#1890ff',
    fontSize: 14,
    borderRadius: 4,
    boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
    colorBgContainer: '#ffffff',
    colorBgLayout: '#f0f2f5',
    colorText: '#262626',
    colorTextSecondary: '#8c8c8c',
    colorTextDisabled: '#bfbfbf',
    colorBorder: '#d9d9d9',
    colorBorderSecondary: '#e8e8e8',
    lineWidth: 1,
    lineType: 'solid',
  },
  components: {
    Button: {
      colorPrimary: '#1890ff',
      colorPrimaryHover: '#40a9ff',
      colorPrimaryActive: '#096dd9',
      borderRadius: 4,
    },
    Input: {
      borderRadius: 4,
      colorBorder: '#d9d9d9',
    },
    Table: {
      colorBorder: '#e8e8e8',
      headerBg: '#fafafa',
      rowHoverBg: '#f5f5f5',
    },
    Card: {
      borderRadius: 4,
      boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
    },
    Menu: {
      colorItemBgHover: '#e6f7ff',
      colorItemTextHover: '#1890ff',
      colorItemText: '#ffffff',
      colorItemBgActive: '#e6f7ff',
      colorBgContainer: '#001529',
      colorText: '#ffffff',
      colorTextSecondary: 'rgba(255, 255, 255, 0.65)',
    },
    Layout: {
      colorBgHeader: '#001529',
      colorBgLayout: '#f0f2f5',
    },
  },
}

export default themeConfig