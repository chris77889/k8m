import React, {useEffect, useRef, useState} from "react";
import {render as amisRender} from "amis";
import {formatFinalGetUrl} from "@/utils/utils";
import {Input, Button, Space} from "@arco-design/web-react";
import {IconSend} from "@arco-design/web-react/icon";

interface WebSocketChatGPTProps {
    url: string;
    params: Record<string, string>;
    data: Record<string, any>;
}

const WebSocketChatGPT = React.forwardRef<HTMLDivElement, WebSocketChatGPTProps>(
    ({url, data, params}, _) => {
        url = formatFinalGetUrl({url, data, params});

        const [messages, setMessages] = useState<{ role: "user" | "ai"; content: string }[]>([]);
        const [status, setStatus] = useState<string>("Disconnected");
        const [inputMessage, setInputMessage] = useState<string>(""); // 用户输入的消息
        const wsRef = useRef<WebSocket | null>(null);
        const messageContainerRef = useRef<HTMLDivElement | null>(null); // 滚动到底部

        useEffect(() => {
            const token = localStorage.getItem("token");
            // 拼接 URL token
            url = url + (url.includes("?") ? "&" : "?") + `token=${token}`;

            const ws = new WebSocket(url);
            wsRef.current = ws;

            ws.onopen = () => setStatus("Connected");

            ws.onmessage = (event) => {
                try {
                    const rawMessage = event.data || "";
                    if (rawMessage) {
                        setMessages((prev) => {
                            if (prev.length === 0 || prev[prev.length - 1].role !== "ai") {
                                // 如果是新的 AI 回复，创建新的条目
                                return [...prev, {role: "ai", content: rawMessage}];
                            } else {
                                // 否则，继续累积在当前 AI 回复中
                                return prev.map((msg, index) =>
                                    index === prev.length - 1 ? {...msg, content: msg.content + rawMessage} : msg
                                );
                            }
                        });
                    }
                } catch (error) {
                    console.error("Failed to parse WebSocket message:", error);
                    setMessages((prev) => [...prev, event.data]);
                }
            };

            ws.onclose = () => setStatus("Disconnected");
            ws.onerror = () => setStatus("Error");

            return () => {
                wsRef.current?.close();
                wsRef.current = null;
            };
        }, [url]);

        // 发送消息
        // 发送消息
        const handleSendMessage = () => {
            if (!inputMessage.trim()) return;

            if (wsRef.current) {
                wsRef.current.send(inputMessage);
            }

            // 立即显示用户消息，并准备新的 AI 回复条目
            setMessages((prev) => [...prev, {role: "user", content: `${inputMessage}`}]);

            setInputMessage(""); // 清空输入框
        };

        // 滚动到底部
        const scrollToBottom = () => {
            if (messageContainerRef.current) {
                messageContainerRef.current.scrollTop = messageContainerRef.current.scrollHeight;
            }
        };

        useEffect(() => {
            scrollToBottom();
        }, [messages]);
        // 监听回车发送消息
        const handleKeyDown = (e: React.KeyboardEvent) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault(); // 防止换行
                handleSendMessage();
            }
        };
        console.log(status)
        return (
            <>
                <div style={{width: "100%", height: "calc(80vh)", minHeight: "600px"}}>
                    <div
                        ref={messageContainerRef}
                        style={{
                            padding: "10px",
                            borderRadius: "5px",
                            overflowY: "auto",
                            display: "flex",
                            flexDirection: "column",
                            gap: "10px",
                        }}
                    >
                        {messages.map((msg, index) => (
                            <div key={index}
                                 style={{
                                     backgroundColor: msg.role === "user" ? "#BFD8FF" : "#EAEAEA", // 用户消息蓝色，AI 消息灰色
                                     color: "#333333", // 文字颜色
                                     padding: "12px",
                                     borderRadius: "8px",
                                     marginBottom: "10px", // 增加间距
                                     maxWidth: "80%", // 限制最大宽度
                                     alignSelf: msg.role === "user" ? "flex-end" : "flex-start", // 用户消息靠右，AI 消息靠左
                                     display: "flex",
                                     flexDirection: "column",
                                 }}
                            >
                                {amisRender({
                                    type: "markdown",
                                    value: msg.content,
                                })}
                            </div>
                        ))}
                    </div>


                </div>
                <div style={{
                    position: "absolute",
                    bottom: "20",
                    width: "90%",
                    display: "flex",
                    alignItems: "center",
                    padding: "8px 12px",
                    borderRadius: "24px",
                    background: "#F5F5F5",
                    border: "1px solid #E0E0E0",
                }}>
                    {/* 左侧图标 */}
                    <Space style={{marginRight: '8px'}}>
                    </Space>

                    {/* 输入框 */}
                    <Input.TextArea
                        style={{
                            flex: 1,
                            border: 'none',
                            outline: 'none',
                            background: 'transparent',
                            resize: 'none',
                        }}
                        value={inputMessage}
                        onChange={setInputMessage}
                        placeholder="请输入消息... Shift+Enter换行，Enter发送"
                        autoSize={{minRows: 2, maxRows: 5}}
                        onKeyDown={handleKeyDown} // 监听回车键
                    />

                    {/* 右侧按钮 */}
                    <Space style={{marginLeft: '8px'}}>
                        <Button
                            type="primary"
                            shape="circle"
                            icon={<IconSend/>}
                            onClick={handleSendMessage}
                            style={{background: 'black', borderColor: 'black'}}
                        />
                    </Space>
                </div>
            </>
        );
    }
);

export default WebSocketChatGPT;