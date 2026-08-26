// Package service — 首次启动向导（Setup Wizard）+ 系统设置业务逻辑。
//
// 向导流程（架构文档 §4.4）：
//   Step 1: 企业信息（公司名/行业/国家）
//   Step 2: 业务场景选择（b2b | b2c | both）
//   Step 3: AI Key 配置（DeepSeek/OpenAI/Qwen，AES 加密入库）
//   Step 4: 修改默认管理员密码
//   Step 5: 标记完成
//
// 完成前：/setup 页可访问，其他页面强制跳转。
// 完成后：/setup 不可访问，正常使用。

package service

import (
	"errors"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/pkg/crypto"
	"github.com/CainGao/trademind/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SetupStatus 首启状态。
type SetupStatus struct {
	Completed        bool     `json:"completed"`
	Steps            []string `json:"steps"`             // 已完成步骤
	CurrentStep      string   `json:"current_step"`      // 下一步
	Scenario         string   `json:"scenario"`          // b2b|b2c|both|（空=未选）
	CompanyConfigured bool    `json:"company_configured"`
	AIKeyConfigured  bool     `json:"ai_key_configured"`
	PasswordChanged  bool     `json:"password_changed"`  // 默认 admin123 是否已改
}

// setup 状态 key（存 settings 表）
const (
	keySetupCompleted    = "setup_completed"
	keySetupCompanyDone  = "setup_company_done"
	keySetupScenarioDone = "setup_scenario_done"
	keySetupAIKeyDone    = "setup_ai_key_done"
	keySetupPwdDone      = "setup_pwd_changed"
	keyScenario          = "scenario" // b2b|b2c|both
	keyModuleB2B         = "module_b2b"
	keyModuleB2C         = "module_b2c"
)

// AI Key 的 settings key
const (
	keyDeepSeekKey = "ai_key_deepseek"
	keyOpenAIKey   = "ai_key_openai"
	keyQwenKey     = "ai_key_qwen"
	keyAnthropicKey = "ai_key_anthropic"
	keyDefaultModel = "ai_default_model"
)

// SetupService 首次启动向导业务。
type SetupService struct {
	companyRepo *repository.CompanyRepo
	settingRepo *repository.SettingRepo
	userRepo    *repository.UserRepo
	encryptor   *crypto.Encryptor
}

// NewSetupService 创建服务。encryptor 用于 AI Key 加密。
func NewSetupService(
	companyRepo *repository.CompanyRepo,
	settingRepo *repository.SettingRepo,
	userRepo *repository.UserRepo,
	encryptor *crypto.Encryptor,
) *SetupService {
	return &SetupService{
		companyRepo: companyRepo,
		settingRepo: settingRepo,
		userRepo:    userRepo,
		encryptor:   encryptor,
	}
}

// GetStatus 读取首启状态。
func (s *SetupService) GetStatus() (*SetupStatus, error) {
	completed := s.isFlagSet(keySetupCompleted)
	companyDone := s.isFlagSet(keySetupCompanyDone)
	scenarioDone := s.isFlagSet(keySetupScenarioDone)
	aiKeyDone := s.isFlagSet(keySetupAIKeyDone)
	pwdDone := s.isFlagSet(keySetupPwdDone)

	scenario, _ := s.settingRepo.Get(keyScenario)
	sc := ""
	if scenario != nil {
		sc = scenario.Value
	}

	steps := []string{}
	if companyDone {
		steps = append(steps, "company")
	}
	if scenarioDone {
		steps = append(steps, "scenario")
	}
	if aiKeyDone {
		steps = append(steps, "ai_key")
	}
	if pwdDone {
		steps = append(steps, "password")
	}

	// 确定下一步
	current := "company"
	if companyDone {
		current = "scenario"
	}
	if scenarioDone {
		current = "ai_key"
	}
	if aiKeyDone {
		current = "password"
	}
	if pwdDone {
		current = "done"
	}

	return &SetupStatus{
		Completed:         completed,
		Steps:             steps,
		CurrentStep:       current,
		Scenario:          sc,
		CompanyConfigured: companyDone,
		AIKeyConfigured:   aiKeyDone,
		PasswordChanged:   pwdDone,
	}, nil
}

func (s *SetupService) isFlagSet(key string) bool {
	s2, err := s.settingRepo.Get(key)
	return err == nil && s2.Value == "true"
}

func (s *SetupService) setFlag(key string) error {
	return s.settingRepo.Set(key, "true", false)
}

// ===== Step 1: 企业信息 =====

// SaveCompanyInput 企业信息入参。
type SaveCompanyInput struct {
	Name     string `json:"name" binding:"required,min=1,max=200"`
	Industry string `json:"industry" binding:"omitempty,max=100"`
	Country  string `json:"country" binding:"omitempty,max=100"`
	Contact  string `json:"contact" binding:"omitempty,max=200"`
}

// SaveCompany 保存企业信息。
func (s *SetupService) SaveCompany(input SaveCompanyInput) error {
	c := &models.Company{
		Name:     input.Name,
		Industry: input.Industry,
		Country:  input.Country,
		Contact:  input.Contact,
	}
	if err := s.companyRepo.Save(c); err != nil {
		return err
	}
	return s.setFlag(keySetupCompanyDone)
}

// ===== Step 2: 业务场景选择 =====

// SelectScenarioInput 场景选择入参。
type SelectScenarioInput struct {
	Scenario string `json:"scenario" binding:"required,oneof=b2b b2c both"`
}

// SelectScenario 选择业务场景，并对应启用模块。
// 架构文档 §4.4: b2b → 启用 b2b 模块；b2c → 启用 b2c；both → 两个都启用。
func (s *SetupService) SelectScenario(input SelectScenarioInput) error {
	enableB2B := input.Scenario == "b2b" || input.Scenario == "both"
	enableB2C := input.Scenario == "b2c" || input.Scenario == "both"

	if err := s.settingRepo.Set(keyScenario, input.Scenario, false); err != nil {
		return err
	}
	if err := s.settingRepo.Set(keyModuleB2B, boolStr(enableB2B), false); err != nil {
		return err
	}
	if err := s.settingRepo.Set(keyModuleB2C, boolStr(enableB2C), false); err != nil {
		return err
	}
	return s.setFlag(keySetupScenarioDone)
}

// ===== Step 3: AI Key 配置 =====

// AIKeyInput AI Key 配置入参。任意一个或多个。
type AIKeyInput struct {
	DeepSeekKey  string `json:"deepseek_key" binding:"omitempty"`
	OpenAIKey    string `json:"openai_key" binding:"omitempty"`
	QwenKey      string `json:"qwen_key" binding:"omitempty"`
	AnthropicKey string `json:"anthropic_key" binding:"omitempty"`
	DefaultModel string `json:"default_model" binding:"omitempty,oneof=deepseek openai qwen anthropic"`
}

// SaveAIKeys 保存 AI Key（AES-256 加密入库，规范 §6.2）。
// 至少配置一个 Key 才算完成。
func (s *SetupService) SaveAIKeys(input AIKeyInput) error {
	hasAny := false
	if input.DeepSeekKey != "" {
		if err := s.saveEncrypted(keyDeepSeekKey, input.DeepSeekKey); err != nil {
			return err
		}
		hasAny = true
	}
	if input.OpenAIKey != "" {
		if err := s.saveEncrypted(keyOpenAIKey, input.OpenAIKey); err != nil {
			return err
		}
		hasAny = true
	}
	if input.QwenKey != "" {
		if err := s.saveEncrypted(keyQwenKey, input.QwenKey); err != nil {
			return err
		}
		hasAny = true
	}
	if input.AnthropicKey != "" {
		if err := s.saveEncrypted(keyAnthropicKey, input.AnthropicKey); err != nil {
			return err
		}
		hasAny = true
	}
	if !hasAny {
		return errors.New("至少配置一个 AI Key")
	}

	// 默认模型
	dm := input.DefaultModel
	if dm == "" {
		// 按配置顺序自动选第一个
		if input.DeepSeekKey != "" {
			dm = "deepseek"
		} else if input.OpenAIKey != "" {
			dm = "openai"
		} else if input.QwenKey != "" {
			dm = "qwen"
		} else {
			dm = "anthropic"
		}
	}
	if err := s.settingRepo.Set(keyDefaultModel, dm, false); err != nil {
		return err
	}

	return s.setFlag(keySetupAIKeyDone)
}

// GetConfiguredAIKeys 返回已配置的 AI 厂商列表（不返回真实 Key）。
func (s *SetupService) GetConfiguredAIKeys() ([]string, error) {
	keys := []string{keyDeepSeekKey, keyOpenAIKey, keyQwenKey, keyAnthropicKey}
	values, err := s.settingRepo.GetMany(keys)
	if err != nil {
		return nil, err
	}
	configured := []string{}
	for _, k := range keys {
		if v, ok := values[k]; ok && v != "" {
			// 脱敏：返回厂商名，不返回 Key
			switch k {
			case keyDeepSeekKey:
				configured = append(configured, "deepseek")
			case keyOpenAIKey:
				configured = append(configured, "openai")
			case keyQwenKey:
				configured = append(configured, "qwen")
			case keyAnthropicKey:
				configured = append(configured, "anthropic")
			}
		}
	}
	return configured, nil
}

// GetDecryptedKey 读取解密后的 AI Key（供 AI 网关调用时用）。
func (s *SetupService) GetDecryptedKey(key string) (string, error) {
	s2, err := s.settingRepo.Get(key)
	if err != nil {
		return "", err
	}
	return s.encryptor.Decrypt(s2.Value)
}

func (s *SetupService) saveEncrypted(key, plaintext string) error {
	encrypted, err := s.encryptor.Encrypt(plaintext)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(key, encrypted, true)
}

// ===== Step 4: 修改默认密码 =====

// ChangePasswordInput 改密码入参。
type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required,min=6"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// ChangePassword 修改管理员密码（规范 §6.1: bcrypt cost=12）。
func (s *SetupService) ChangePassword(userID uint, input ChangePasswordInput) error {
	// 新旧密码不得相同（防「改了个寂寞」绕过默认密码治理，gotcha #88）
	if input.NewPassword == input.OldPassword {
		return errors.New("新密码不能与原密码相同")
	}
	// 弱密码黑名单拦截（防把密码改回 admin123 后照样标记首启完成）
	if isWeakPassword(input.NewPassword) {
		return errors.New("新密码过于简单（命中常见弱密码黑名单），请更换")
	}
	u, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(input.OldPassword)); err != nil {
		return errors.New("原密码错误")
	}
	hash, err := crypto.HashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	if err := s.userRepo.ChangePassword(userID, hash); err != nil {
		return err
	}
	return s.setFlag(keySetupPwdDone)
}

// ===== Step 5: 标记完成 =====

// Complete 标记首启向导完成。
// 必须所有前置步骤完成才能调用。
func (s *SetupService) Complete() error {
	status, err := s.GetStatus()
	if err != nil {
		return err
	}
	if !status.CompanyConfigured {
		return errors.New("请先完成企业信息配置")
	}
	if !status.AIKeyConfigured {
		return errors.New("请先配置 AI Key")
	}
	if !status.PasswordChanged {
		return errors.New("请先修改默认密码")
	}
	return s.setFlag(keySetupCompleted)
}

// IsSetupRequired 是否需要首启（用于路由守卫）。
func (s *SetupService) IsSetupRequired() bool {
	return !s.isFlagSet(keySetupCompleted)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ErrSetupRequired 首启未完成（供中间件判断）。
var ErrSetupRequired = errors.New("setup_required")

// 兼容 gorm 错误判断（避免上层再引 gorm）
var _ = gorm.ErrRecordNotFound
