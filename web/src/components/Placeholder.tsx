// 通用占位页面：用于尚未实现的模块路由占位。
import { Result, Button } from "antd";
import type { ReactNode } from "react";

interface PlaceholderProps {
  title: string;
  subtitle?: string;
  icon?: ReactNode;
}

export default function Placeholder({ title, subtitle, icon }: PlaceholderProps) {
  return (
    <Result
      icon={icon}
      status="info"
      title={title}
      subTitle={subtitle || "该模块正在开发中，敬请期待"}
      extra={<Button type="primary">查看路线图</Button>}
    />
  );
}
