package main

import (
    "encoding/json"
    "fmt"
    "html/template" 
    "io/ioutil"
    "log"
    "net/http"
    neturl "net/url"
    "os"
    "strconv"
    "strings"
	"path/filepath"

    "github.com/gorilla/mux"
    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
)



// 配置结构
type Config struct {
    Port         string `json:"port"`
    AdminPwd     string `json:"admin_password"` // 存储为bcrypt哈希
    DataFile     string `json:"data_file"`
    SiteTitle    string `json:"site_title"`    // 站点标题
    SiteFooter   string `json:"site_footer"`   // 页脚自定义文本
    IcpNumber    string `json:"icp_number"`
    IcpLink      string `json:"icp_link"`
    PoliceNumber string `json:"police_number"`
    PoliceLink   string `json:"police_link"`
}


// 书签链接结构
type BookmarkLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Desc string `json:"desc"`
	Icon string `json:"icon,omitempty"`
}

// 书签分类结构
type BookmarkCategory struct {
	Category string         `json:"category"`
	Icon     string         `json:"icon"`
	Links    []BookmarkLink `json:"links"`
}

// 全局变量
var (
	config     Config
	bookmarks  []BookmarkCategory
	store      *sessions.CookieStore
	dataDir    = "data"                                   
    configFile = filepath.Join(dataDir, "config.json")   

)

// 初始化函数
func init() {
    // 确保data目录存在
    if _, err := os.Stat(dataDir); os.IsNotExist(err) {
        if err := os.MkdirAll(dataDir, 0755); err != nil {
            log.Fatalf("无法创建数据目录: %v", err)
        }
        log.Printf("创建数据目录: %s", dataDir)
    }
}


func main() {
	// 加载配置
	log.Printf("使用数据目录: %s", dataDir)
    log.Printf("配置文件路径: %s", configFile)

	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化会话存储，使用随机密钥
	key := []byte("super-secret-key-change-in-production")
	store = sessions.NewCookieStore(key)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7天
		HttpOnly: true,
	}

	// 加载书签数据
	if err := loadBookmarks(); err != nil {
		log.Fatalf("加载书签数据失败: %v", err)
	}

	// 设置路由
	r := mux.NewRouter()
	
	// 静态文件服务
	staticFs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", staticFs))
	imageFs := http.FileServer(http.Dir("./images"))
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", imageFs))
	
	// 公共页面
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/api/bookmarks", getBookmarksHandler).Methods("GET")
    r.HandleFunc("/api/favicon", getFaviconHandler).Methods("GET")

	// 管理员页面
	r.HandleFunc("/admin", adminLoginHandler).Methods("GET")
	r.HandleFunc("/admin/login", adminLoginPostHandler).Methods("POST")
	r.HandleFunc("/admin/logout", adminLogoutHandler).Methods("GET")
	
	// 受保护的管理员API（需要认证）
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(authMiddleware)
	admin.HandleFunc("/dashboard", adminDashboardHandler).Methods("GET")
	admin.HandleFunc("/api/bookmarks", updateBookmarksHandler).Methods("POST")
	admin.HandleFunc("/api/category", addCategoryHandler).Methods("POST")
	admin.HandleFunc("/api/category/{category}", updateCategoryHandler).Methods("PUT") // 新增
	admin.HandleFunc("/api/category/{category}", deleteCategoryHandler).Methods("DELETE")
	admin.HandleFunc("/api/bookmark", addBookmarkHandler).Methods("POST")
	admin.HandleFunc("/api/bookmark/{category}/{index}", updateBookmarkHandler).Methods("PUT") // 新增
	admin.HandleFunc("/api/bookmark/{category}/{index}", deleteBookmarkHandler).Methods("DELETE")
	admin.HandleFunc("/api/change-password", changePasswordHandler).Methods("POST")
    admin.HandleFunc("/api/site-settings", getSiteSettingsHandler).Methods("GET")
    admin.HandleFunc("/api/site-settings", updateSiteSettingsHandler).Methods("POST")
    admin.HandleFunc("/api/bookmarks", updateAllBookmarksHandler).Methods("POST")

	// 启动服务器
	port := config.Port
	if port == "" {
		port = "1239"
	}
	
	fmt.Printf("服务器运行在 http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// 加载配置
func loadConfig() error {
    // 检查配置文件是否存在
    if _, err := os.Stat(configFile); os.IsNotExist(err) {
        // 创建默认配置
        defaultPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
        config = Config{
            Port:     "1239",
            AdminPwd: string(defaultPassword),
            DataFile: filepath.Join(dataDir, "bookmarks.json"), 
        }
        
        // 保存默认配置
        return saveConfig()
    }
    
    // 读取配置文件
    data, err := ioutil.ReadFile(configFile)
    if err != nil {
        return err
    }
    
    // 解析配置
    if err := json.Unmarshal(data, &config); err != nil {
        return err
    }
    
    // 确保数据文件路径正确（兼容旧配置）
    if !strings.HasPrefix(config.DataFile, dataDir) {
        config.DataFile = filepath.Join(dataDir, filepath.Base(config.DataFile))
        saveConfig()  // 保存修改后的配置
    }
    
    return nil
}


// 保存配置
func saveConfig() error {
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return err
    }
    
    return ioutil.WriteFile(configFile, data, 0644)
}


// 加载书签
func loadBookmarks() error {
    log.Printf("尝试从 %s 加载书签数据", config.DataFile)
    // 检查数据文件是否存在
    if _, err := os.Stat(config.DataFile); os.IsNotExist(err) {
        // 创建默认书签数据
        bookmarks = []BookmarkCategory{
            {
                Category: "常用网站",
                Icon:     "🔥",
                Links: []BookmarkLink{
                    {
                        Name: "百度",
                        URL:  "https://www.baidu.com",
                        Desc: "全球最大的中文搜索引擎",
                        Icon: "https://www.google.com/s2/favicons?domain=baidu.com",
                    },
                    {
                        Name: "腾讯网",
                        URL:  "https://www.qq.com",
                        Desc: "新闻资讯门户网站",
                        Icon: "https://www.google.com/s2/favicons?domain=qq.com",
                    },
                },
            },
            {
                Category: "新闻资讯",
                Icon:     "📰",
                Links: []BookmarkLink{
                    {
                        Name: "新浪新闻",
                        URL:  "https://news.sina.com.cn/",
                        Desc: "新浪网新闻中心",
                        Icon: "https://www.google.com/s2/favicons?domain=sina.com.cn",
                    },
                    {
                        Name: "网易",
                        URL:  "https://www.163.com/",
                        Desc: "领先的互联网技术公司",
                        Icon: "https://www.google.com/s2/favicons?domain=163.com",
                    },
                },
            },
        }
        
        // 保存默认书签
        return saveBookmarks()
    }
    
    // 读取数据文件
    data, err := ioutil.ReadFile(config.DataFile)
    if err != nil {
        return err
    }
    
    return json.Unmarshal(data, &bookmarks)
}


// 保存书签
func saveBookmarks() error {
    // 确保数据文件目录存在
    if _, err := os.Stat(filepath.Dir(config.DataFile)); os.IsNotExist(err) {
        if err := os.MkdirAll(filepath.Dir(config.DataFile), 0755); err != nil {
            return err
        }
    }
    
    data, err := json.MarshalIndent(bookmarks, "", "  ")
    if err != nil {
        return err
    }
    
    return ioutil.WriteFile(config.DataFile, data, 0644)
}


// 首页处理
func homeHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("templates/index.html")
    if err != nil {
        http.Error(w, "模板加载失败", http.StatusInternalServerError)
        return
    }
    
    data := struct {
        SiteTitle    string
        SiteFooter   string
        IcpNumber    string
        IcpLink      string
        PoliceNumber string
        PoliceLink   string
    }{
        SiteTitle:    config.SiteTitle,
        SiteFooter:   config.SiteFooter,
        IcpNumber:    config.IcpNumber,
        IcpLink:      config.IcpLink,
        PoliceNumber: config.PoliceNumber,
        PoliceLink:   config.PoliceLink,
    }
    
    tmpl.Execute(w, data)
}



// 获取书签API
func getBookmarksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookmarks)
}

// 管理员登录页面
func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
    // 检查是否已经登录
    session, _ := store.Get(r, "admin-session")
    if auth, ok := session.Values["authenticated"].(bool); ok && auth {
        http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
        return
    }
    
    // 使用html/template包中的函数
    tmpl, err := template.ParseFiles("templates/admin_login.html")
    if err != nil {
        http.Error(w, "模板加载失败", http.StatusInternalServerError)
        return
    }
    
    tmpl.Execute(w, nil)
}


// 管理员登录处理
func adminLoginPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	password := r.FormValue("password")
	
	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(config.AdminPwd), []byte(password)); err != nil {
		http.Redirect(w, r, "/admin?error=1", http.StatusSeeOther)
		return
	}
	
	// 创建会话
	session, _ := store.Get(r, "admin-session")
	session.Values["authenticated"] = true
	session.Save(r, w)
	
	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// 管理员注销
func adminLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "admin-session")
	session.Values["authenticated"] = false
	session.Save(r, w)
	
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// 管理员仪表盘
func adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("templates/admin_dashboard.html")
    if err != nil {
        http.Error(w, "模板加载失败", http.StatusInternalServerError)
        return
    }
    
    data := struct {
        Bookmarks []BookmarkCategory
    }{
        Bookmarks: bookmarks,
    }
    
    tmpl.Execute(w, data)
}


// 认证中间件
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "admin-session")
		
		// 检查是否认证
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// 更新书签处理
func updateBookmarksHandler(w http.ResponseWriter, r *http.Request) {
	var newBookmarks []BookmarkCategory
	
	// 解析JSON请求
	err := json.NewDecoder(r.Body).Decode(&newBookmarks)
	if err != nil {
		http.Error(w, "无效的JSON数据", http.StatusBadRequest)
		return
	}
	
	// 更新书签数据
	bookmarks = newBookmarks
	
	// 保存到文件
	if err := saveBookmarks(); err != nil {
		http.Error(w, "保存书签失败", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 添加分类处理
func addCategoryHandler(w http.ResponseWriter, r *http.Request) {
	var category BookmarkCategory
	
	// 解析JSON请求
	err := json.NewDecoder(r.Body).Decode(&category)
	if err != nil {
		http.Error(w, "无效的JSON数据", http.StatusBadRequest)
		return
	}
	
	// 添加新分类
	bookmarks = append(bookmarks, category)
	
	// 保存到文件
	if err := saveBookmarks(); err != nil {
		http.Error(w, "保存书签失败", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 更新分类处理 - 新增
func updateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oldCategory := vars["category"]
	
	var categoryUpdate struct {
		NewCategory string `json:"newCategory"`
		Icon        string `json:"icon"`
	}
	
	// 解析JSON请求
	err := json.NewDecoder(r.Body).Decode(&categoryUpdate)
	if err != nil {
		http.Error(w, "无效的JSON数据", http.StatusBadRequest)
		return
	}
	
	// 查找分类并更新
	found := false
	for i, cat := range bookmarks {
		if cat.Category == oldCategory {
			bookmarks[i].Category = categoryUpdate.NewCategory
			bookmarks[i].Icon = categoryUpdate.Icon
			found = true
			break
		}
	}
	
	if !found {
		http.Error(w, "未找到指定分类", http.StatusNotFound)
		return
	}
	
	// 保存到文件
	if err := saveBookmarks(); err != nil {
		http.Error(w, "保存书签失败", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 删除分类处理
func deleteCategoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]
	
	// 查找并删除分类
	for i, cat := range bookmarks {
		if cat.Category == category {
			bookmarks = append(bookmarks[:i], bookmarks[i+1:]...)
			break
		}
	}
	
	// 保存到文件
	if err := saveBookmarks(); err != nil {
		http.Error(w, "保存书签失败", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 添加书签处理
func addBookmarkHandler(w http.ResponseWriter, r *http.Request) {
    var data struct {
        Category string      `json:"category"`
        Bookmark BookmarkLink `json:"bookmark"`
    }
    
    // 解析JSON请求
    err := json.NewDecoder(r.Body).Decode(&data)
    if err != nil {
        http.Error(w, "无效的JSON数据", http.StatusBadRequest)
        return
    }
    
    // 检查图标是否为空，如果为空则自动获取favicon URL
    if data.Bookmark.Icon == "" {
        faviconURL, err := getFavicon(data.Bookmark.URL)
        if err == nil {
            // 设置为favicon URL
            data.Bookmark.Icon = faviconURL
        }
    }
    
    // 查找分类并添加书签
    found := false
    for i, cat := range bookmarks {
        if cat.Category == data.Category {
            bookmarks[i].Links = append(bookmarks[i].Links, data.Bookmark)
            found = true
            break
        }
    }
    
    if !found {
        http.Error(w, "未找到指定分类", http.StatusNotFound)
        return
    }
    
    // 保存到文件
    if err := saveBookmarks(); err != nil {
        http.Error(w, "保存书签失败", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}


// 更新书签处理
func updateBookmarkHandler(w http.ResponseWriter, r *http.Request) {    
    vars := mux.Vars(r)
    category := vars["category"]
    indexStr := vars["index"]
    
    // 将索引转换为整数
    idx, err := strconv.Atoi(indexStr)
    if err != nil {
        http.Error(w, "无效的索引", http.StatusBadRequest)
        return
    }
    
    var updatedBookmark BookmarkLink
    
    // 解析JSON请求
    err = json.NewDecoder(r.Body).Decode(&updatedBookmark)
    if err != nil {
        http.Error(w, "无效的JSON数据", http.StatusBadRequest)
        return
    }
    
    // 检查图标是否为空，如果为空则自动获取favicon URL
    if updatedBookmark.Icon == "" {
        faviconURL, err := getFavicon(updatedBookmark.URL)
        if err == nil {
            // 设置为favicon URL
            updatedBookmark.Icon = faviconURL
        }
    }
    
    // 查找分类并更新书签
    found := false
    for i, cat := range bookmarks {
        if cat.Category == category {
            if idx >= 0 && idx < len(cat.Links) {
                bookmarks[i].Links[idx] = updatedBookmark
                found = true
                break
            }
        }
    }
    
    if !found {
        http.Error(w, "未找到指定书签", http.StatusNotFound)
        return
    }
    
    // 保存到文件
    if err := saveBookmarks(); err != nil {
        http.Error(w, "保存书签失败", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 删除书签处理
func deleteBookmarkHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category := vars["category"]
	indexStr := vars["index"]
	
	// 将索引转换为整数
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		http.Error(w, "无效的索引", http.StatusBadRequest)
		return
	}
	
	// 查找分类并删除书签
	for i, cat := range bookmarks {
		if cat.Category == category && idx >= 0 && idx < len(cat.Links) {
			bookmarks[i].Links = append(cat.Links[:idx], cat.Links[idx+1:]...)
			break
		}
	}
	
	// 保存到文件
	if err := saveBookmarks(); err != nil {
		http.Error(w, "保存书签失败", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 获取网站favicon的函数
func getFavicon(siteURL string) (string, error) {
    // 确保URL格式正确
    if !strings.HasPrefix(siteURL, "http://") && !strings.HasPrefix(siteURL, "https://") {
        siteURL = "https://" + siteURL
    }
    
    // 解析URL获取域名
    parsedURL, err := neturl.Parse(siteURL)
    if err != nil {
        return "", err
    }
    
    // 构建Google的favicon服务URL
    faviconURL := "https://www.google.com/s2/favicons?domain=" + parsedURL.Hostname() + "&sz=64"
    
    return faviconURL, nil
}

// 处理密码修改的函数
func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
    // 解析请求体
    var passwordData struct {
        CurrentPassword string `json:"currentPassword"`
        NewPassword     string `json:"newPassword"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&passwordData); err != nil {
        http.Error(w, "无效的请求数据", http.StatusBadRequest)
        return
    }
    
    // 验证当前密码
    if err := bcrypt.CompareHashAndPassword([]byte(config.AdminPwd), []byte(passwordData.CurrentPassword)); err != nil {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "status": "error",
            "message": "当前密码错误",
        })
        return
    }
    
    // 生成新密码的哈希值
    newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(passwordData.NewPassword), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "密码哈希生成失败", http.StatusInternalServerError)
        return
    }
    
    // 更新配置中的密码哈希
    config.AdminPwd = string(newPasswordHash)
    
    // 保存配置
    if err := saveConfig(); err != nil {
        http.Error(w, "保存配置失败", http.StatusInternalServerError)
        return
    }
    
    // 返回成功消息
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "message": "密码修改成功",
    })
}

// favicon处理器
func getFaviconHandler(w http.ResponseWriter, r *http.Request) {
    // 获取查询参数中的URL
    urlParam := r.URL.Query().Get("url")
    if urlParam == "" {
        http.Error(w, "缺少url参数", http.StatusBadRequest)
        return
    }
    
    // 使用现有的getFavicon函数获取favicon URL
    faviconURL, err := getFavicon(urlParam)
    if err != nil {
        http.Error(w, "获取favicon失败", http.StatusInternalServerError)
        return
    }
    
    // 返回favicon URL
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(faviconURL))
}

// 获取站点设置
func getSiteSettingsHandler(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "status": "success",
        "data": map[string]string{
            "title":        config.SiteTitle,
            "footer":       config.SiteFooter,
            "icpNumber":    config.IcpNumber,
            "icpLink":      config.IcpLink,
            "policeNumber": config.PoliceNumber,
            "policeLink":   config.PoliceLink,
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// 更新站点设置
func updateSiteSettingsHandler(w http.ResponseWriter, r *http.Request) {
    var settings struct {
        Title        string `json:"title"`
        Footer       string `json:"footer"`
        IcpNumber    string `json:"icpNumber"`
        IcpLink      string `json:"icpLink"`
        PoliceNumber string `json:"policeNumber"`
        PoliceLink   string `json:"policeLink"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
        http.Error(w, "无效的请求数据", http.StatusBadRequest)
        return
    }
    
    // 更新配置
    config.SiteTitle = settings.Title
    config.SiteFooter = settings.Footer
    config.IcpNumber = settings.IcpNumber
    config.IcpLink = settings.IcpLink
    config.PoliceNumber = settings.PoliceNumber
    config.PoliceLink = settings.PoliceLink
    
    // 保存配置
    if err := saveConfig(); err != nil {
        http.Error(w, "保存配置失败", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "message": "站点设置已更新",
    })
}

// 处理整个书签数据更新
func updateAllBookmarksHandler(w http.ResponseWriter, r *http.Request) {
    // 解析请求体
    var updatedBookmarks []BookmarkCategory
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&updatedBookmarks); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // 更新书签数据
    bookmarks = updatedBookmarks
    
    // 保存到文件
    if err := saveBookmarks(); err != nil {
        http.Error(w, "Failed to save bookmarks", http.StatusInternalServerError)
        return
    }
    
    // 返回成功响应
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
    })
}
