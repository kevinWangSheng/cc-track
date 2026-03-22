// Generator 模板：新增测试时复制此模板填空

package {{.Package}}

import (
	"testing"
)

func Test{{.FuncName}}(t *testing.T) {
	tests := []struct {
		name    string
		// input fields
		want    interface{}
		wantErr bool
	}{
		{
			name:    "{{.Scenario1}}",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "{{.ErrorScenario}}",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// db := newTestDB(t)  // 如果需要 DB

			// Act
			// got, err := {{.FuncName}}(...)

			// Assert
			// if (err != nil) != tt.wantErr {
			// 	t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			// }
			// if got != tt.want {
			// 	t.Errorf("got %v, want %v", got, tt.want)
			// }
		})
	}
}
